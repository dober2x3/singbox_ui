package prober

import (
	"fmt"
	"log"
	"net"
	"sync"
	"syscall"
	"time"

	"singbox-config-service/internal/pkg/types"
)

// probeLoop is the main loop that periodically probes all nodes.
func (p *Prober) probeLoop() {
	defer p.wg.Done()

	p.probeAllNodes()

	ticker := time.NewTicker(time.Duration(p.config.Interval) * time.Second)
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

// probeAllNodes probes every registered node concurrently, limited by the semaphore.
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

// probeNode performs a TCP probe against a single node, with retries on failure.
func (p *Prober) probeNode(node types.ProbeNode) {
	var latency int64 = -1
	var success bool

	maxRetries := p.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

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

// tcpProbe attempts a TCP connection to the given address and port.
// If BindAddress or BindInterface is configured, the probe bypasses the
// system routing table (e.g. a TUN tunnel) and goes through the specified
// local IP or network interface instead.
func (p *Prober) tcpProbe(address string, port int) bool {
	addr := fmt.Sprintf("%s:%d", address, port)

	dialer := &net.Dialer{
		Timeout: time.Duration(p.config.Timeout) * time.Millisecond,
	}

	// Bind to a specific local IP address if configured.
	// This makes the TCP connection originate from that IP, bypassing
	// any TUN tunnel that might otherwise capture the traffic.
	if p.config.BindAddress != "" {
		localAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(p.config.BindAddress, "0"))
		if err == nil {
			dialer.LocalAddr = localAddr
		}
	}

	// Bind to a specific network interface if configured.
	// Uses SO_BINDTODEVICE (Linux only; requires root or CAP_NET_ADMIN).
	// Forces the socket to egress through the named physical interface
	// (e.g. "eth0") even when a TUN interface is the default route.
	if p.config.BindInterface != "" {
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				if err := syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, soBindToDevice, p.config.BindInterface); err != nil {
					log.Printf("Prober: failed to bind to interface %s: %v", p.config.BindInterface, err)
				}
			})
		}
	}

	conn, err := dialer.DialContext(p.ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// updateResult persists the probe result and computes the new status.
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
