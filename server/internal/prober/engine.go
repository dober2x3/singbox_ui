package prober

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"singbox-config-service/internal/pkg/types"
)

type nodeHistory struct {
	mu      sync.Mutex
	results []bool
	index   int
	size    int
}

func (h *nodeHistory) update(success bool) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.results[h.index] = success
	h.index = (h.index + 1) % h.size

	successCount := 0
	for _, r := range h.results {
		if r {
			successCount++
		}
	}
	return float64(successCount) / float64(h.size) * 100
}

const maxRetries = 2

type Prober struct {
	config    ProberConfig
	nodes     sync.Map // map[string]types.ProbeNode
	results   sync.Map // map[string]*types.ProbeResult
	history   sync.Map // map[string]*nodeHistory
	running   int32
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	semaphore chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewProber(config ProberConfig) *Prober {
	ctx, cancel := context.WithCancel(context.Background())
	return &Prober{
		config:    config,
		stopChan:  make(chan struct{}),
		semaphore: make(chan struct{}, config.ProbeConcurrent),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func DefaultProberConfig() ProberConfig {
	return ProberConfig{
		ProbeInterval:   30,
		ProbeTimeout:    5000,
		ProbeConcurrent: 5,
		MaxResults:      100,
	}
}

func (p *Prober) Start() {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return
	}

	p.mu.Lock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	log.Println("Prober started")

	p.wg.Add(1)
	go p.probeLoop()
}

func (p *Prober) Stop() {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return
	}

	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
	}
	close(p.stopChan)
	p.mu.Unlock()

	p.wg.Wait()
	log.Println("Prober stopped")
}

func (p *Prober) IsRunning() bool {
	return atomic.LoadInt32(&p.running) == 1
}

func (p *Prober) AddNode(node types.ProbeNode) {
	p.nodes.Store(node.Tag, node)

	result := &types.ProbeResult{
		NodeTag:   node.Tag,
		Protocol:  node.Protocol,
		Address:   node.Address,
		Port:      node.Port,
		Latency:   -1,
		Status:    "unknown",
		LastProbe: "",
	}
	p.results.Store(node.Tag, result)

	history := &nodeHistory{
		results: make([]bool, p.config.MaxResults),
		index:   0,
		size:    p.config.MaxResults,
	}
	p.history.Store(node.Tag, history)

	log.Printf("Prober: added node %s (%s://%s:%d)", node.Tag, node.Protocol, node.Address, node.Port)
}

func (p *Prober) RemoveNode(tag string) {
	p.nodes.Delete(tag)
	p.results.Delete(tag)
	p.history.Delete(tag)
	log.Printf("Prober: removed node %s", tag)
}

func (p *Prober) ClearNodes() {
	p.nodes.Range(func(key, _ interface{}) bool {
		p.nodes.Delete(key)
		return true
	})
	p.results.Range(func(key, _ interface{}) bool {
		p.results.Delete(key)
		return true
	})
	p.history.Range(func(key, _ interface{}) bool {
		p.history.Delete(key)
		return true
	})
	log.Println("Prober: cleared all nodes")
}

func (p *Prober) UpdateNodes(nodes []types.ProbeNode) {
	p.ClearNodes()
	for _, node := range nodes {
		p.AddNode(node)
	}
	log.Printf("Prober: updated with %d nodes", len(nodes))
}

func (p *Prober) GetResult(tag string) *types.ProbeResult {
	if result, ok := p.results.Load(tag); ok {
		r := result.(*types.ProbeResult)
		copy := *r
		return &copy
	}
	return nil
}

func (p *Prober) GetAllResults() map[string]*types.ProbeResult {
	results := make(map[string]*types.ProbeResult)
	p.results.Range(func(key, value interface{}) bool {
		r := value.(*types.ProbeResult)
		copy := *r
		results[key.(string)] = &copy
		return true
	})
	return results
}

func (p *Prober) GetBestNode() *types.ProbeResult {
	var best *types.ProbeResult
	p.results.Range(func(_, value interface{}) bool {
		result := value.(*types.ProbeResult)
		if result.Status == "online" && result.Latency > 0 {
			if best == nil || result.Latency < best.Latency {
				copy := *result
				best = &copy
			}
		}
		return true
	})
	return best
}

func (p *Prober) GetOnlineNodes() []*types.ProbeResult {
	var online []*types.ProbeResult
	p.results.Range(func(_, value interface{}) bool {
		result := value.(*types.ProbeResult)
		if result.Status == "online" {
			copy := *result
			online = append(online, &copy)
		}
		return true
	})
	return online
}

func (p *Prober) probeLoop() {
	defer p.wg.Done()

	p.probeAllNodes()

	ticker := time.NewTicker(time.Duration(p.config.ProbeInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.probeAllNodes()
		}
	}
}

func (p *Prober) probeAllNodes() {
	var wg sync.WaitGroup

	p.nodes.Range(func(_, value interface{}) bool {
		if !p.IsRunning() {
			return false
		}

		node := value.(types.ProbeNode)

		wg.Add(1)
		go func(n types.ProbeNode) {
			defer wg.Done()

			select {
			case p.semaphore <- struct{}{}:
				defer func() { <-p.semaphore }()
				p.probeNode(n)
			case <-p.ctx.Done():
				return
			}
		}(node)

		return true
	})

	wg.Wait()
}

func (p *Prober) probeNode(node types.ProbeNode) {
	var latency int64 = -1
	var success bool

	for retry := 0; retry <= maxRetries; retry++ {
		if !p.IsRunning() {
			return
		}

		if retry > 0 {
			select {
			case <-time.After(time.Duration(retry*500) * time.Millisecond):
			case <-p.ctx.Done():
				return
			}
		}

		start := time.Now()
		success = p.tcpProbe(node.Address, node.Port)

		if success {
			latency = time.Since(start).Milliseconds()
			break
		}
	}

	p.updateResult(node.Tag, latency, success)
}

func (p *Prober) tcpProbe(address string, port int) bool {
	addr := fmt.Sprintf("%s:%d", address, port)

	dialer := &net.Dialer{
		Timeout: time.Duration(p.config.ProbeTimeout) * time.Millisecond,
	}

	conn, err := dialer.DialContext(p.ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (p *Prober) updateResult(tag string, latency int64, success bool) {
	resultVal, ok := p.results.Load(tag)
	if !ok {
		return
	}
	result := resultVal.(*types.ProbeResult)

	historyVal, ok := p.history.Load(tag)
	if !ok {
		return
	}
	history := historyVal.(*nodeHistory)

	successRate := history.update(success)

	newResult := &types.ProbeResult{
		NodeTag:     result.NodeTag,
		Protocol:    result.Protocol,
		Address:     result.Address,
		Port:        result.Port,
		Latency:     latency,
		LastProbe:   time.Now().Format("2006-01-02 15:04:05"),
		SuccessRate: successRate,
	}

	if success {
		newResult.Status = "online"
		newResult.FailCount = 0
	} else {
		newResult.FailCount = result.FailCount + 1
		if newResult.FailCount >= 3 {
			newResult.Status = "offline"
		} else {
			newResult.Status = "timeout"
		}
	}

	p.results.Store(tag, newResult)
}

func (p *Prober) GetStats() map[string]interface{} {
	var totalNodes, onlineNodes, offlineNodes, timeoutNodes int

	p.results.Range(func(_, value interface{}) bool {
		result := value.(*types.ProbeResult)
		totalNodes++
		switch result.Status {
		case "online":
			onlineNodes++
		case "offline":
			offlineNodes++
		case "timeout":
			timeoutNodes++
		}
		return true
	})

	return map[string]interface{}{
		"running":      p.IsRunning(),
		"totalNodes":   totalNodes,
		"onlineNodes":  onlineNodes,
		"offlineNodes": offlineNodes,
		"timeoutNodes": timeoutNodes,
		"config": map[string]interface{}{
			"probeInterval":   p.config.ProbeInterval,
			"probeTimeout":    p.config.ProbeTimeout,
			"probeConcurrent": p.config.ProbeConcurrent,
			"maxResults":      p.config.MaxResults,
		},
	}
}

func (p *Prober) SaveNodesToFile(baseDir string) error {
	var nodes []types.ProbeNode
	p.nodes.Range(func(_, value interface{}) bool {
		nodes = append(nodes, value.(types.ProbeNode))
		return true
	})

	data, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(baseDir, "prober_nodes.json")
	return os.WriteFile(filePath, data, 0644)
}

func (p *Prober) LoadNodesFromFile(baseDir string) error {
	filePath := filepath.Join(baseDir, "prober_nodes.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var nodes []types.ProbeNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return err
	}

	p.UpdateNodes(nodes)
	return nil
}
