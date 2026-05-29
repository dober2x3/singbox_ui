package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// ProbeResult probe result for a single node
type ProbeResult struct {
	NodeTag     string    `json:"nodeTag"`
	Protocol    string    `json:"protocol"`
	Address     string    `json:"address"`
	Port        int       `json:"port"`
	Latency     int64     `json:"latency"`     // latency (ms), -1 means timeout/failure
	Status      string    `json:"status"`      // "online" | "offline" | "timeout" | "unknown"
	LastProbe   time.Time `json:"lastProbe"`   // last probe time
	FailCount   int       `json:"failCount"`   // consecutive failure count
	SuccessRate float64   `json:"successRate"` // success rate (0-100)
}

// ProberConfig prober configuration
type ProberConfig struct {
	ProbeInterval   time.Duration `json:"probeInterval"`   // probe interval
	ProbeTimeout    time.Duration `json:"probeTimeout"`    // single probe timeout
	MaxRetries      int           `json:"maxRetries"`      // max retry count
	MaxConcurrent   int           `json:"maxConcurrent"`   // max concurrent probes
	ProbeURL        string        `json:"probeURL"`        // HTTP probe URL
	HistorySize     int           `json:"historySize"`     // history size (for calculating success rate)
	EnableTCPProbe  bool          `json:"enableTCPProbe"`  // enable TCP probe
	EnableHTTPProbe bool          `json:"enableHTTPProbe"` // enable HTTP probe
}

// ProbeNode node to be probed
type ProbeNode struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

// nodeHistory node history (for calculating success rate) - thread safe
type nodeHistory struct {
	mu      sync.Mutex
	results []bool // true = success, false = failure
	index   int    // ring buffer index
	size    int    // history size
}

// update updates history and returns success rate
func (h *nodeHistory) update(success bool) float64 {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.results[h.index] = success
	h.index = (h.index + 1) % h.size

	// Calculate success rate
	successCount := 0
	for _, r := range h.results {
		if r {
			successCount++
		}
	}
	return float64(successCount) / float64(h.size) * 100
}

// Prober async high-frequency prober
type Prober struct {
	config     ProberConfig
	nodes      sync.Map // map[string]ProbeNode
	results    sync.Map // map[string]*ProbeResult
	history    sync.Map // map[string]*nodeHistory
	running    int32    // atomic ops, 0=stopped, 1=running
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex // protects stopChan creation and closing
	httpClient *http.Client
	semaphore  chan struct{} // concurrency control semaphore
	ctx        context.Context
	cancel     context.CancelFunc
}

// DefaultProberConfig default prober configuration
func DefaultProberConfig() ProberConfig {
	return ProberConfig{
		ProbeInterval:   30 * time.Second,
		ProbeTimeout:    5 * time.Second,
		MaxRetries:      2,
		MaxConcurrent:   10,
		ProbeURL:        "http://www.google.com/generate_204",
		HistorySize:     10,
		EnableTCPProbe:  true,
		EnableHTTPProbe: false, // default only TCP probe, HTTP probe is optional
	}
}

// Global prober instance
var (
	globalProber *Prober
	proberMutex  sync.RWMutex
)

// NewProber creates a new prober instance
func NewProber(config ProberConfig) *Prober {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Prober{
		config:    config,
		stopChan:  make(chan struct{}),
		semaphore: make(chan struct{}, config.MaxConcurrent),
		ctx:       ctx,
		cancel:    cancel,
		httpClient: &http.Client{
			Timeout: config.ProbeTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				DialContext: (&net.Dialer{
					Timeout:   config.ProbeTimeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
	return p
}

// InitProber initializes the global prober
func InitProber() error {
	proberMutex.Lock()
	defer proberMutex.Unlock()

	if globalProber != nil {
		return nil
	}

	config := DefaultProberConfig()

	// Read config from environment variables
	if interval := os.Getenv("PROBER_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.ProbeInterval = d
		}
	}
	if timeout := os.Getenv("PROBER_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.ProbeTimeout = d
		}
	}

	globalProber = NewProber(config)
	log.Printf("Prober initialized with interval=%v, timeout=%v", config.ProbeInterval, config.ProbeTimeout)
	return nil
}

// GetProber gets the global prober instance
func GetProber() *Prober {
	proberMutex.RLock()
	defer proberMutex.RUnlock()
	return globalProber
}

// Start starts the prober
func (p *Prober) Start() {
	// Use atomic operation to check and set running state
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return // already running
	}

	p.mu.Lock()
	// Recreate context and stopChan
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.stopChan = make(chan struct{})
	p.mu.Unlock()

	log.Println("Prober started")

	p.wg.Add(1)
	go p.probeLoop()
}

// Stop stops the prober
func (p *Prober) Stop() {
	// Use atomic operation to check and set running state
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return // already stopped
	}

	p.mu.Lock()
	// Cancel context
	if p.cancel != nil {
		p.cancel()
	}
	// Close stopChan
	close(p.stopChan)
	p.mu.Unlock()

	// Wait for probe loop to finish
	p.wg.Wait()

	// Close HTTP client idle connections
	if transport, ok := p.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	log.Println("Prober stopped")
}

// IsRunning checks if the prober is running
func (p *Prober) IsRunning() bool {
	return atomic.LoadInt32(&p.running) == 1
}

// AddNode adds a node to probe
func (p *Prober) AddNode(node ProbeNode) {
	p.nodes.Store(node.Tag, node)

	// Initialize result
	result := &ProbeResult{
		NodeTag:   node.Tag,
		Protocol:  node.Protocol,
		Address:   node.Address,
		Port:      node.Port,
		Latency:   -1,
		Status:    "unknown",
		LastProbe: time.Time{},
	}
	p.results.Store(node.Tag, result)

	// Initialize history
	history := &nodeHistory{
		results: make([]bool, p.config.HistorySize),
		index:   0,
		size:    p.config.HistorySize,
	}
	p.history.Store(node.Tag, history)

	log.Printf("Prober: added node %s (%s://%s:%d)", node.Tag, node.Protocol, node.Address, node.Port)
}

// RemoveNode removes a node
func (p *Prober) RemoveNode(tag string) {
	p.nodes.Delete(tag)
	p.results.Delete(tag)
	p.history.Delete(tag)
	log.Printf("Prober: removed node %s", tag)
}

// ClearNodes clears all nodes
func (p *Prober) ClearNodes() {
	p.nodes.Range(func(key, value interface{}) bool {
		p.nodes.Delete(key)
		return true
	})
	p.results.Range(func(key, value interface{}) bool {
		p.results.Delete(key)
		return true
	})
	p.history.Range(func(key, value interface{}) bool {
		p.history.Delete(key)
		return true
	})
	log.Println("Prober: cleared all nodes")
}

// UpdateNodes batch update nodes (replaces all existing nodes)
func (p *Prober) UpdateNodes(nodes []ProbeNode) {
	p.ClearNodes()
	for _, node := range nodes {
		p.AddNode(node)
	}
	log.Printf("Prober: updated with %d nodes", len(nodes))
}

// GetResult gets a single node's probe result (returns copy to avoid race)
func (p *Prober) GetResult(tag string) *ProbeResult {
	if result, ok := p.results.Load(tag); ok {
		r := result.(*ProbeResult)
		// Return a copy
		return &ProbeResult{
			NodeTag:     r.NodeTag,
			Protocol:    r.Protocol,
			Address:     r.Address,
			Port:        r.Port,
			Latency:     r.Latency,
			Status:      r.Status,
			LastProbe:   r.LastProbe,
			FailCount:   r.FailCount,
			SuccessRate: r.SuccessRate,
		}
	}
	return nil
}

// GetAllResults gets all nodes' probe results
func (p *Prober) GetAllResults() map[string]*ProbeResult {
	results := make(map[string]*ProbeResult)
	p.results.Range(func(key, value interface{}) bool {
		r := value.(*ProbeResult)
		// Return a copy
		results[key.(string)] = &ProbeResult{
			NodeTag:     r.NodeTag,
			Protocol:    r.Protocol,
			Address:     r.Address,
			Port:        r.Port,
			Latency:     r.Latency,
			Status:      r.Status,
			LastProbe:   r.LastProbe,
			FailCount:   r.FailCount,
			SuccessRate: r.SuccessRate,
		}
		return true
	})
	return results
}

// GetBestNode gets the lowest latency online node
func (p *Prober) GetBestNode() *ProbeResult {
	var best *ProbeResult
	p.results.Range(func(key, value interface{}) bool {
		result := value.(*ProbeResult)
		if result.Status == "online" && result.Latency > 0 {
			if best == nil || result.Latency < best.Latency {
				// Copy result
				best = &ProbeResult{
					NodeTag:     result.NodeTag,
					Protocol:    result.Protocol,
					Address:     result.Address,
					Port:        result.Port,
					Latency:     result.Latency,
					Status:      result.Status,
					LastProbe:   result.LastProbe,
					FailCount:   result.FailCount,
					SuccessRate: result.SuccessRate,
				}
			}
		}
		return true
	})
	return best
}

// GetOnlineNodes gets all online nodes
func (p *Prober) GetOnlineNodes() []*ProbeResult {
	var online []*ProbeResult
	p.results.Range(func(key, value interface{}) bool {
		result := value.(*ProbeResult)
		if result.Status == "online" {
			// Return a copy
			online = append(online, &ProbeResult{
				NodeTag:     result.NodeTag,
				Protocol:    result.Protocol,
				Address:     result.Address,
				Port:        result.Port,
				Latency:     result.Latency,
				Status:      result.Status,
				LastProbe:   result.LastProbe,
				FailCount:   result.FailCount,
				SuccessRate: result.SuccessRate,
			})
		}
		return true
	})
	return online
}

// probeLoop probe loop
func (p *Prober) probeLoop() {
	defer p.wg.Done()

	// Execute probe immediately once
	p.probeAllNodes()

	ticker := time.NewTicker(p.config.ProbeInterval)
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

// probeAllNodes concurrently probes all nodes
func (p *Prober) probeAllNodes() {
	var wg sync.WaitGroup

	p.nodes.Range(func(key, value interface{}) bool {
		// Check if stopped
		if !p.IsRunning() {
			return false
		}

		node := value.(ProbeNode)

		wg.Add(1)
		go func(n ProbeNode) {
			defer wg.Done()

			// Use context for timeout control and cancellation
			select {
			case p.semaphore <- struct{}{}:
				// Acquired semaphore
				defer func() { <-p.semaphore }()
				p.probeNode(n)
			case <-p.ctx.Done():
				// Prober stopped
				return
			}
		}(node)

		return true
	})

	wg.Wait()
}

// probeNode probes a single node (with retry)
func (p *Prober) probeNode(node ProbeNode) {
	var latency int64 = -1
	var success bool

	for retry := 0; retry <= p.config.MaxRetries; retry++ {
		// Check if stopped
		if !p.IsRunning() {
			return
		}

		if retry > 0 {
			// Use cancellable sleep
			select {
			case <-time.After(time.Duration(retry*500) * time.Millisecond):
			case <-p.ctx.Done():
				return
			}
		}

		start := time.Now()

		if p.config.EnableTCPProbe {
			success = p.tcpProbe(node.Address, node.Port)
		} else if p.config.EnableHTTPProbe {
			success = p.httpProbe()
		} else {
			success = p.tcpProbe(node.Address, node.Port)
		}

		if success {
			latency = time.Since(start).Milliseconds()
			break
		}
	}

	// Update result
	p.updateResult(node.Tag, latency, success)
}

// tcpProbe TCP connection probe (supports context cancellation)
func (p *Prober) tcpProbe(address string, port int) bool {
	addr := fmt.Sprintf("%s:%d", address, port)

	// Use context-aware Dialer
	dialer := &net.Dialer{
		Timeout: p.config.ProbeTimeout,
	}

	conn, err := dialer.DialContext(p.ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// httpProbe HTTP probe
func (p *Prober) httpProbe() bool {
	ctx, cancel := context.WithTimeout(p.ctx, p.config.ProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", p.config.ProbeURL, nil)
	if err != nil {
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Read and discard response body, ensure connection can be reused
	io.Copy(io.Discard, resp.Body)

	// 200 or 204 are both considered success
	return resp.StatusCode == 200 || resp.StatusCode == 204
}

// updateResult updates probe result
func (p *Prober) updateResult(tag string, latency int64, success bool) {
	resultVal, ok := p.results.Load(tag)
	if !ok {
		return
	}
	result := resultVal.(*ProbeResult)

	// Get history
	historyVal, ok := p.history.Load(tag)
	if !ok {
		// Node may have been deleted
		return
	}
	history := historyVal.(*nodeHistory)

	// Thread-safe update history and get success rate
	successRate := history.update(success)

	// Create new result object (avoid direct modification)
	newResult := &ProbeResult{
		NodeTag:     result.NodeTag,
		Protocol:    result.Protocol,
		Address:     result.Address,
		Port:        result.Port,
		Latency:     latency,
		LastProbe:   time.Now(),
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

// GetStats gets prober statistics
func (p *Prober) GetStats() map[string]interface{} {
	var totalNodes, onlineNodes, offlineNodes, timeoutNodes int

	p.results.Range(func(key, value interface{}) bool {
		result := value.(*ProbeResult)
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
			"probeInterval": p.config.ProbeInterval.String(),
			"probeTimeout":  p.config.ProbeTimeout.String(),
			"maxRetries":    p.config.MaxRetries,
			"maxConcurrent": p.config.MaxConcurrent,
		},
	}
}

// SaveNodesToFile saves node config to file
func (p *Prober) SaveNodesToFile() error {
	var nodes []ProbeNode
	p.nodes.Range(func(key, value interface{}) bool {
		nodes = append(nodes, value.(ProbeNode))
		return true
	})

	data, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(singboxDir, "prober_nodes.json")
	return os.WriteFile(filePath, data, 0644)
}

// LoadNodesFromFile loads node config from file
func (p *Prober) LoadNodesFromFile() error {
	filePath := filepath.Join(singboxDir, "prober_nodes.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // file not existing is not an error
		}
		return err
	}

	var nodes []ProbeNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return err
	}

	p.UpdateNodes(nodes)
	return nil
}

// StopProber stops the global prober
func StopProber() {
	proberMutex.Lock()
	defer proberMutex.Unlock()

	if globalProber != nil {
		globalProber.Stop()
		globalProber = nil
	}
}
