package prober

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"singbox-config-service/internal/pkg/types"
)

// TestNewProber verifies that NewProber initialises correctly.
func TestNewProber(t *testing.T) {
	p := NewProber(DefaultConfig())
	if p == nil {
		t.Fatal("NewProber() returned nil")
	}

	if p.config.Interval != 30 {
		t.Errorf("Expected Interval 30, got %d", p.config.Interval)
	}

	if p.config.Timeout != 5000 {
		t.Errorf("Expected Timeout 5000, got %d", p.config.Timeout)
	}

	if p.ctx == nil {
		t.Error("Context should not be nil")
	}
	if p.cancel == nil {
		t.Error("Cancel func should not be nil")
	}
}

// TestProberAddRemoveNode verifies adding and removing nodes.
func TestProberAddRemoveNode(t *testing.T) {
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{
		Tag:      "test-node-1",
		Protocol: "vmess",
		Address:  "127.0.0.1",
		Port:     10086,
	})

	result := p.GetResult("test-node-1")
	if result == nil {
		t.Fatal("Node not found after adding")
	}

	if result.NodeTag != "test-node-1" {
		t.Errorf("Expected tag 'test-node-1', got '%s'", result.NodeTag)
	}

	if result.Status != "unknown" {
		t.Errorf("Expected status 'unknown', got '%s'", result.Status)
	}

	p.RemoveNode("test-node-1")
	result = p.GetResult("test-node-1")
	if result != nil {
		t.Error("Node should not exist after removal")
	}
}

// TestProberUpdateNodes verifies that UpdateNodes replaces all nodes.
func TestProberUpdateNodes(t *testing.T) {
	p := NewProber(DefaultConfig())

	nodes := []types.ProbeNode{
		{Tag: "node-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443},
		{Tag: "node-2", Protocol: "vless", Address: "2.2.2.2", Port: 443},
		{Tag: "node-3", Protocol: "trojan", Address: "3.3.3.3", Port: 443},
	}
	p.UpdateNodes(nodes)

	results := p.GetAllResults()
	if len(results) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(results))
	}

	for _, node := range nodes {
		result := p.GetResult(node.Tag)
		if result == nil {
			t.Errorf("Node %s not found", node.Tag)
		}
	}

	newNodes := []types.ProbeNode{
		{Tag: "new-node-1", Protocol: "ss", Address: "4.4.4.4", Port: 8388},
	}
	p.UpdateNodes(newNodes)

	results = p.GetAllResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 node after update, got %d", len(results))
	}

	if p.GetResult("node-1") != nil {
		t.Error("Old node should not exist after update")
	}
}

// TestProberClearNodes verifies that ClearNodes removes all nodes.
func TestProberClearNodes(t *testing.T) {
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{Tag: "a", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	p.AddNode(types.ProbeNode{Tag: "b", Protocol: "vmess", Address: "2.2.2.2", Port: 443})

	p.ClearNodes()

	results := p.GetAllResults()
	if len(results) != 0 {
		t.Errorf("Expected 0 nodes after clear, got %d", len(results))
	}
}

// TestProberStartStop verifies the start/stop lifecycle and idempotency.
func TestProberStartStop(t *testing.T) {
	p := NewProber(DefaultConfig())

	if p.IsRunning() {
		t.Error("Prober should not be running initially")
	}

	if atomic.LoadInt32(&p.running) != 0 {
		t.Error("Initial running state should be 0")
	}

	p.Start()
	if !p.IsRunning() {
		t.Error("Prober should be running after Start()")
	}

	p.Start()
	if !p.IsRunning() {
		t.Error("Prober should still be running after repeated Start()")
	}

	p.Stop()
	if p.IsRunning() {
		t.Error("Prober should not be running after Stop()")
	}

	p.Stop()
	if p.IsRunning() {
		t.Error("Prober should still be stopped after repeated Stop()")
	}
}

// TestProberStartStopRace verifies concurrent start/stop does not panic.
func TestProberStartStopRace(t *testing.T) {
	p := NewProber(DefaultConfig())

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.Start()
		}()
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			p.Stop()
		}()
	}
	wg.Wait()

	p.Stop()
}

// TestProberGetBestNode verifies best-node selection by latency.
func TestProberGetBestNode(t *testing.T) {
	p := NewProber(DefaultConfig())

	best := p.GetBestNode()
	if best != nil {
		t.Error("GetBestNode() should return nil when no nodes")
	}

	p.AddNode(types.ProbeNode{Tag: "slow", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	p.AddNode(types.ProbeNode{Tag: "fast", Protocol: "vmess", Address: "2.2.2.2", Port: 443})
	p.AddNode(types.ProbeNode{Tag: "offline", Protocol: "vmess", Address: "3.3.3.3", Port: 443})

	p.results.Store("slow", &types.ProbeResult{
		NodeTag: "slow", Latency: 200, Status: "online",
	})
	p.results.Store("fast", &types.ProbeResult{
		NodeTag: "fast", Latency: 50, Status: "online",
	})
	p.results.Store("offline", &types.ProbeResult{
		NodeTag: "offline", Latency: -1, Status: "offline",
	})

	best = p.GetBestNode()
	if best == nil {
		t.Fatal("GetBestNode returned nil")
	}

	if best.NodeTag != "fast" {
		t.Errorf("Expected best node 'fast', got '%s'", best.NodeTag)
	}
}

// TestProberGetOnlineNodes verifies online node filtering.
func TestProberGetOnlineNodes(t *testing.T) {
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{Tag: "online-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	p.AddNode(types.ProbeNode{Tag: "online-2", Protocol: "vmess", Address: "2.2.2.2", Port: 443})
	p.AddNode(types.ProbeNode{Tag: "offline", Protocol: "vmess", Address: "3.3.3.3", Port: 443})

	p.results.Store("online-1", &types.ProbeResult{NodeTag: "online-1", Status: "online"})
	p.results.Store("online-2", &types.ProbeResult{NodeTag: "online-2", Status: "online"})
	p.results.Store("offline", &types.ProbeResult{NodeTag: "offline", Status: "offline"})

	online := p.GetOnlineNodes()
	if len(online) != 2 {
		t.Errorf("Expected 2 online nodes, got %d", len(online))
	}
}

// TestProberConcurrentAccess verifies thread-safe concurrent node operations.
func TestProberConcurrentAccess(t *testing.T) {
	p := NewProber(DefaultConfig())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p.AddNode(types.ProbeNode{
				Tag:      "node-" + string(rune('0'+i%10)) + string(rune('0'+i/10)),
				Protocol: "vmess",
				Address:  "1.1.1.1",
				Port:     10000 + i,
			})
		}(i)
	}
	wg.Wait()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.GetAllResults()
			p.GetBestNode()
			p.GetOnlineNodes()
		}()
	}
	wg.Wait()
}

// TestProberStats verifies the stats output counts correctly.
func TestProberStats(t *testing.T) {
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{Tag: "node-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	p.AddNode(types.ProbeNode{Tag: "node-2", Protocol: "vmess", Address: "2.2.2.2", Port: 443})
	p.results.Store("node-1", &types.ProbeResult{NodeTag: "node-1", Status: "online"})
	p.results.Store("node-2", &types.ProbeResult{NodeTag: "node-2", Status: "timeout"})

	stats := p.GetStats()

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

// TestProberUpdateResult verifies result status transitions based on success/failure.
func TestProberUpdateResult(t *testing.T) {
	p := NewProber(DefaultConfig())
	p.config.MaxResults = 5

	p.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})

	for i := 0; i < 5; i++ {
		p.updateResult("test", 100, true)
	}

	result := p.GetResult("test")
	if result.Status != "online" {
		t.Errorf("Expected status 'online', got '%s'", result.Status)
	}
	if result.SuccessRate != 100 {
		t.Errorf("Expected success rate 100, got %v", result.SuccessRate)
	}
	if result.FailCount != 0 {
		t.Errorf("Expected fail count 0, got %d", result.FailCount)
	}

	for i := 0; i < 3; i++ {
		p.updateResult("test", -1, false)
	}

	result = p.GetResult("test")
	if result.Status != "offline" {
		t.Errorf("Expected status 'offline' after 3 failures, got '%s'", result.Status)
	}
	if result.FailCount != 3 {
		t.Errorf("Expected fail count 3, got %d", result.FailCount)
	}
}

// TestProberUpdateResultDeletedNode verifies no panic when updating a deleted node.
func TestProberUpdateResultDeletedNode(t *testing.T) {
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	p.RemoveNode("test")

	p.updateResult("test", 100, true)
}

// TestProberResultIsCopy verifies that GetResult returns a copy not the original.
func TestProberResultIsCopy(t *testing.T) {
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})
	p.results.Store("test", &types.ProbeResult{
		NodeTag: "test",
		Status:  "online",
		Latency: 100,
	})

	result1 := p.GetResult("test")
	result2 := p.GetResult("test")

	result1.Latency = 999

	if result2.Latency == 999 {
		t.Error("GetResult should return a copy, not the original")
	}
}

// TestProberContextCancellation verifies clean stop within a reasonable time.
func TestProberContextCancellation(t *testing.T) {
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "192.0.2.1", Port: 443})

	p.Start()

	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		p.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("Stop should not block for too long")
	}
}

// TestNodeHistoryThreadSafe verifies thread-safe concurrent history updates.
func TestNodeHistoryThreadSafe(t *testing.T) {
	h := &nodeHistory{
		results: make([]bool, 10),
		index:   0,
		size:    10,
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h.update(i%2 == 0)
		}(i)
	}
	wg.Wait()
}

// TestSaveNodesToFile verifies saving and loading nodes to/from a file.
func TestSaveNodesToFile(t *testing.T) {
	dir := t.TempDir()
	p := NewProber(DefaultConfig())

	p.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})

	if err := p.SaveNodesToFile(dir); err != nil {
		t.Fatalf("SaveNodesToFile() error = %v", err)
	}

	p.ClearNodes()
	if len(p.GetAllResults()) != 0 {
		t.Error("ClearNodes() did not clear")
	}

	if err := p.LoadNodesFromFile(dir); err != nil {
		t.Fatalf("LoadNodesFromFile() error = %v", err)
	}

	results := p.GetAllResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 node after load, got %d", len(results))
	}

	result := p.GetResult("test")
	if result == nil {
		t.Fatal("Loaded node 'test' not found")
	}
	if result.Protocol != "vmess" {
		t.Errorf("Expected protocol 'vmess', got '%s'", result.Protocol)
	}
}

// TestSaveNodesToFile_NoFile verifies LoadNodesFromFile handles a missing file gracefully.
func TestSaveNodesToFile_NoFile(t *testing.T) {
	dir := t.TempDir()
	p := NewProber(DefaultConfig())

	if err := p.LoadNodesFromFile(dir); err != nil {
		t.Errorf("LoadNodesFromFile() should not error when file does not exist: %v", err)
	}
}
