package services

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewProber(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	if prober == nil {
		t.Fatal("NewProber returned nil")
	}

	if prober.config.ProbeInterval != 30*time.Second {
		t.Errorf("Expected ProbeInterval 30s, got %v", prober.config.ProbeInterval)
	}

	if prober.config.ProbeTimeout != 5*time.Second {
		t.Errorf("Expected ProbeTimeout 5s, got %v", prober.config.ProbeTimeout)
	}

	// Verify context was created
	if prober.ctx == nil {
		t.Error("Context should not be nil")
	}
	if prober.cancel == nil {
		t.Error("Cancel func should not be nil")
	}
}

func TestProberAddRemoveNode(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	// Add node
	node := ProbeNode{
		Tag:      "test-node-1",
		Protocol: "vmess",
		Address:  "127.0.0.1",
		Port:     10086,
	}
	prober.AddNode(node)

	// Verify node was added
	result := prober.GetResult("test-node-1")
	if result == nil {
		t.Fatal("Node not found after adding")
	}

	if result.NodeTag != "test-node-1" {
		t.Errorf("Expected tag 'test-node-1', got '%s'", result.NodeTag)
	}

	if result.Status != "unknown" {
		t.Errorf("Expected status 'unknown', got '%s'", result.Status)
	}

	// Remove node
	prober.RemoveNode("test-node-1")

	result = prober.GetResult("test-node-1")
	if result != nil {
		t.Error("Node should not exist after removal")
	}
}

func TestProberUpdateNodes(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	// Batch add nodes
	nodes := []ProbeNode{
		{Tag: "node-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443},
		{Tag: "node-2", Protocol: "vless", Address: "2.2.2.2", Port: 443},
		{Tag: "node-3", Protocol: "trojan", Address: "3.3.3.3", Port: 443},
	}

	prober.UpdateNodes(nodes)

	results := prober.GetAllResults()
	if len(results) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(results))
	}

	// Verify each node
	for _, node := range nodes {
		result := prober.GetResult(node.Tag)
		if result == nil {
			t.Errorf("Node %s not found", node.Tag)
		}
	}

	// Update to new node set (should clear old nodes)
	newNodes := []ProbeNode{
		{Tag: "new-node-1", Protocol: "ss", Address: "4.4.4.4", Port: 8388},
	}
	prober.UpdateNodes(newNodes)

	results = prober.GetAllResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 node after update, got %d", len(results))
	}

	// Old node should not exist
	if prober.GetResult("node-1") != nil {
		t.Error("Old node should not exist after update")
	}
}

func TestProberClearNodes(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	// Add nodes
	nodes := []ProbeNode{
		{Tag: "node-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443},
		{Tag: "node-2", Protocol: "vless", Address: "2.2.2.2", Port: 443},
	}
	prober.UpdateNodes(nodes)

	// Clear all
	prober.ClearNodes()

	results := prober.GetAllResults()
	if len(results) != 0 {
		t.Errorf("Expected 0 nodes after clear, got %d", len(results))
	}
}

func TestProberStartStop(t *testing.T) {
	config := DefaultProberConfig()
	config.ProbeInterval = 100 * time.Millisecond // Fast test
	prober := NewProber(config)

	// Verify initial state
	if prober.IsRunning() {
		t.Error("Prober should not be running initially")
	}

	// Verify running is int32 (atomic operation)
	if atomic.LoadInt32(&prober.running) != 0 {
		t.Error("Initial running state should be 0")
	}

	// Start
	prober.Start()

	if !prober.IsRunning() {
		t.Error("Prober should be running after Start()")
	}

	// Repeated start should be no-op
	prober.Start()
	if !prober.IsRunning() {
		t.Error("Prober should still be running")
	}

	// Stop
	prober.Stop()

	if prober.IsRunning() {
		t.Error("Prober should not be running after Stop()")
	}

	// Repeated stop should be no-op
	prober.Stop()
	if prober.IsRunning() {
		t.Error("Prober should still be stopped")
	}
}

func TestProberStartStopRace(t *testing.T) {
	config := DefaultProberConfig()
	config.ProbeInterval = 50 * time.Millisecond
	prober := NewProber(config)

	// Concurrent start and stop
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			prober.Start()
		}()
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			prober.Stop()
		}()
	}
	wg.Wait()

	// Ensure final state is consistent
	prober.Stop() // Ensure stopped
}

func TestProberTCPProbe(t *testing.T) {
	config := DefaultProberConfig()
	config.ProbeTimeout = 2 * time.Second
	prober := NewProber(config)

	// Test reachable address (Google DNS)
	success := prober.tcpProbe("8.8.8.8", 53)
	if !success {
		t.Log("Warning: TCP probe to 8.8.8.8:53 failed (may be network issue)")
	}

	// Test unreachable address
	success = prober.tcpProbe("192.0.2.1", 12345) // Documentation reserved address
	if success {
		t.Error("TCP probe to unreachable address should fail")
	}
}

func TestProberGetBestNode(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	// Add nodes and simulate results
	prober.AddNode(ProbeNode{Tag: "slow", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	prober.AddNode(ProbeNode{Tag: "fast", Protocol: "vmess", Address: "2.2.2.2", Port: 443})
	prober.AddNode(ProbeNode{Tag: "offline", Protocol: "vmess", Address: "3.3.3.3", Port: 443})

	// Manually set results
	prober.results.Store("slow", &ProbeResult{
		NodeTag: "slow",
		Latency: 200,
		Status:  "online",
	})
	prober.results.Store("fast", &ProbeResult{
		NodeTag: "fast",
		Latency: 50,
		Status:  "online",
	})
	prober.results.Store("offline", &ProbeResult{
		NodeTag: "offline",
		Latency: -1,
		Status:  "offline",
	})

	best := prober.GetBestNode()
	if best == nil {
		t.Fatal("GetBestNode returned nil")
	}

	if best.NodeTag != "fast" {
		t.Errorf("Expected best node 'fast', got '%s'", best.NodeTag)
	}
}

func TestProberGetOnlineNodes(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	// Add nodes and simulate results
	prober.AddNode(ProbeNode{Tag: "online-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	prober.AddNode(ProbeNode{Tag: "online-2", Protocol: "vmess", Address: "2.2.2.2", Port: 443})
	prober.AddNode(ProbeNode{Tag: "offline", Protocol: "vmess", Address: "3.3.3.3", Port: 443})

	prober.results.Store("online-1", &ProbeResult{NodeTag: "online-1", Status: "online"})
	prober.results.Store("online-2", &ProbeResult{NodeTag: "online-2", Status: "online"})
	prober.results.Store("offline", &ProbeResult{NodeTag: "offline", Status: "offline"})

	online := prober.GetOnlineNodes()
	if len(online) != 2 {
		t.Errorf("Expected 2 online nodes, got %d", len(online))
	}
}

func TestProberConcurrentAccess(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	// Concurrently add nodes
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			prober.AddNode(ProbeNode{
				Tag:      "node-" + string(rune('0'+i%10)) + string(rune('0'+i/10)),
				Protocol: "vmess",
				Address:  "1.1.1.1",
				Port:     10000 + i,
			})
		}(i)
	}
	wg.Wait()

	// Concurrently read
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			prober.GetAllResults()
			prober.GetBestNode()
			prober.GetOnlineNodes()
		}()
	}
	wg.Wait()

	// If no panic, test passes
	t.Log("Concurrent access test passed")
}

func TestProberStats(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	prober.AddNode(ProbeNode{Tag: "node-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	prober.AddNode(ProbeNode{Tag: "node-2", Protocol: "vmess", Address: "2.2.2.2", Port: 443})
	prober.results.Store("node-1", &ProbeResult{NodeTag: "node-1", Status: "online"})
	prober.results.Store("node-2", &ProbeResult{NodeTag: "node-2", Status: "timeout"})

	stats := prober.GetStats()

	if stats["totalNodes"].(int) != 2 {
		t.Errorf("Expected totalNodes 2, got %v", stats["totalNodes"])
	}

	if stats["onlineNodes"].(int) != 1 {
		t.Errorf("Expected onlineNodes 1, got %v", stats["onlineNodes"])
	}

	if stats["timeoutNodes"].(int) != 1 {
		t.Errorf("Expected timeoutNodes 1, got %v", stats["timeoutNodes"])
	}

	if stats["running"].(bool) != false {
		t.Errorf("Expected running false, got %v", stats["running"])
	}
}

func TestNodeHistoryThreadSafe(t *testing.T) {
	h := &nodeHistory{
		results: make([]bool, 10),
		index:   0,
		size:    10,
	}

	// Concurrently update history
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h.update(i%2 == 0)
		}(i)
	}
	wg.Wait()

	// If no panic or data race, test passes
	t.Log("NodeHistory thread safety test passed")
}

func TestProberUpdateResult(t *testing.T) {
	config := DefaultProberConfig()
	config.HistorySize = 5
	prober := NewProber(config)

	prober.AddNode(ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})

	// Simulate consecutive successes
	for i := 0; i < 5; i++ {
		prober.updateResult("test", 100, true)
	}

	result := prober.GetResult("test")
	if result.Status != "online" {
		t.Errorf("Expected status 'online', got '%s'", result.Status)
	}
	if result.SuccessRate != 100 {
		t.Errorf("Expected success rate 100, got %v", result.SuccessRate)
	}
	if result.FailCount != 0 {
		t.Errorf("Expected fail count 0, got %d", result.FailCount)
	}

	// Simulate consecutive failures
	for i := 0; i < 3; i++ {
		prober.updateResult("test", -1, false)
	}

	result = prober.GetResult("test")
	if result.Status != "offline" {
		t.Errorf("Expected status 'offline' after 3 failures, got '%s'", result.Status)
	}
	if result.FailCount != 3 {
		t.Errorf("Expected fail count 3, got %d", result.FailCount)
	}
}

func TestProberUpdateResultDeletedNode(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	prober.AddNode(ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})

	// Remove node
	prober.RemoveNode("test")

	// Updating a deleted node should not panic
	prober.updateResult("test", 100, true)
}

func TestProberResultIsCopy(t *testing.T) {
	config := DefaultProberConfig()
	prober := NewProber(config)

	prober.AddNode(ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	prober.results.Store("test", &ProbeResult{
		NodeTag: "test",
		Status:  "online",
		Latency: 100,
	})

	// Get results
	result1 := prober.GetResult("test")
	result2 := prober.GetResult("test")

	// Modify result1
	result1.Latency = 999

	// result2 should not be affected
	if result2.Latency == 999 {
		t.Error("GetResult should return a copy, not the original")
	}
}

func TestProberContextCancellation(t *testing.T) {
	config := DefaultProberConfig()
	config.ProbeInterval = 100 * time.Millisecond
	prober := NewProber(config)

	prober.AddNode(ProbeNode{Tag: "test", Protocol: "vmess", Address: "192.0.2.1", Port: 443})

	prober.Start()

	// Wait briefly for probing to start
	time.Sleep(50 * time.Millisecond)

	// Stop prober
	done := make(chan struct{})
	go func() {
		prober.Stop()
		close(done)
	}()

	// Verify Stop does not block too long
	select {
	case <-done:
		// Normal stop
	case <-time.After(3 * time.Second):
		t.Error("Stop should not block for too long")
	}
}
