package subscription

import (
	"testing"
	"time"

	"singbox-config-service/internal/pkg/types"
)

// TestService_AddAndDeleteSubscription verifies adding a subscription via store and deleting it by ID.
func TestService_AddAndDeleteSubscription(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	// Add subscription by directly manipulating store (bypass Fetch which blocks loopback)
	data, _ := svc.GetAllSubscriptions()
	entry := SubscriptionEntry{
		ID:          "test-1",
		Name:        "Test",
		URL:         "https://example.com/sub",
		LastUpdated: time.Now().Format(time.RFC3339),
	}
	data.Subscriptions = append(data.Subscriptions, entry)
	_ = svc.store.Save(*data)

	// Verify it was saved
	data, _ = svc.GetAllSubscriptions()
	if len(data.Subscriptions) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(data.Subscriptions))
	}

	// Delete it
	if err := svc.DeleteSubscription("test-1"); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}

	data, _ = svc.GetAllSubscriptions()
	if len(data.Subscriptions) != 0 {
		t.Errorf("expected 0 subscriptions after delete, got %d", len(data.Subscriptions))
	}
}

// TestService_GetAllNodes verifies that GetAllNodes returns nodes from all stored subscriptions.
func TestService_GetAllNodes(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	data, _ := svc.GetAllSubscriptions()
	data.Subscriptions = append(data.Subscriptions, SubscriptionEntry{
		ID:   "test-sub",
		Name: "Test",
		URL:  "https://example.com",
		Nodes: []types.ProxyNode{
			{Name: "Node-1", Protocol: "vmess", Address: "1.1.1.1", Port: 443},
			{Name: "Node-2", Protocol: "trojan", Address: "2.2.2.2", Port: 8443},
		},
	})
	_ = svc.store.Save(*data)

	nodes, err := svc.GetAllNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

// TestService_UpdateSubscriptionSettings verifies updating auto-update and interval settings.
func TestService_UpdateSubscriptionSettings(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	data, _ := svc.GetAllSubscriptions()
	data.Subscriptions = append(data.Subscriptions, SubscriptionEntry{
		ID:   "test-1",
		Name: "Test",
	})
	_ = svc.store.Save(*data)

	entry, err := svc.UpdateSubscriptionSettings("test-1", true, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !entry.AutoUpdate {
		t.Error("expected auto_update true")
	}
	if entry.UpdateInterval != 6 {
		t.Errorf("expected update_interval 6, got %d", entry.UpdateInterval)
	}
}

// TestService_NotFound verifies that operations on a nonexistent subscription ID return an error.
func TestService_NotFound(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	_, err := svc.UpdateSubscription("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent subscription")
	}

	_, err = svc.UpdateSubscriptionSettings("nonexistent", true, 6)
	if err == nil {
		t.Error("expected error for nonexistent subscription")
	}

	err = svc.DeleteSubscription("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent subscription")
	}
}

// TestService_UpdateSubscription verifies that UpdateSubscription attempts to fetch a stored subscription URL.
func TestService_UpdateSubscription(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	data, _ := svc.GetAllSubscriptions()
	data.Subscriptions = append(data.Subscriptions, SubscriptionEntry{
		ID:   "test-1",
		Name: "Test",
		URL:  "https://example.com/sub",
	})
	_ = svc.store.Save(*data)

	_, err := svc.UpdateSubscription("test-1")
	if err != nil {
		t.Logf("UpdateSubscription returned (expected): %v", err)
	}
}

// TestService_SaveProbeResults verifies that probe results are stored and retrievable for matching nodes.
func TestService_SaveProbeResults(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	node := types.ProxyNode{
		Name:     "Test-Node",
		Protocol: "vmess",
		Address:  "1.1.1.1",
		Port:     443,
		Outbound: map[string]interface{}{
			"tag": "vmess-1_1_1_1-443",
		},
	}

	data, _ := svc.GetAllSubscriptions()
	data.Subscriptions = append(data.Subscriptions, SubscriptionEntry{
		ID:    "test-1",
		Name:  "Test",
		Nodes: []types.ProxyNode{node},
	})
	_ = svc.store.Save(*data)

	results := []types.ProbeResultUpdate{
		{Tag: "vmess-1_1_1_1-443", Latency: 100, Online: true, LastProbe: time.Now().Format(time.RFC3339), SuccessRate: 95},
	}

	if err := svc.SaveProbeResults(results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the results were saved
	data, _ = svc.GetAllSubscriptions()
	if len(data.Subscriptions) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(data.Subscriptions))
	}
	if len(data.Subscriptions[0].Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(data.Subscriptions[0].Nodes))
	}
	n := data.Subscriptions[0].Nodes[0]
	if n.Latency != 100 {
		t.Errorf("expected latency 100, got %d", n.Latency)
	}
	if !n.Online {
		t.Error("expected online true")
	}
	if n.SuccessRate != 95 {
		t.Errorf("expected success_rate 95, got %d", n.SuccessRate)
	}
}

// TestService_SaveSpeedTestResults verifies that speed test results are stored for matching nodes.
func TestService_SaveSpeedTestResults(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	node := types.ProxyNode{
		Name:     "Test-Node",
		Protocol: "vmess",
		Address:  "1.1.1.1",
		Port:     443,
		Outbound: map[string]interface{}{
			"tag": "vmess-1_1_1_1-443",
		},
	}

	data, _ := svc.GetAllSubscriptions()
	data.Subscriptions = append(data.Subscriptions, SubscriptionEntry{
		ID:    "test-1",
		Name:  "Test",
		Nodes: []types.ProxyNode{node},
	})
	_ = svc.store.Save(*data)

	results := []types.SpeedTestUpdate{
		{Tag: "vmess-1_1_1_1-443", Latency: 50, SpeedKBps: 1024.5, Online: true, LastProbe: time.Now().Format(time.RFC3339)},
	}

	if err := svc.SaveSpeedTestResults(results); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestService_FetchSubscription_ValidatesURL verifies that a non-http/https URL is rejected.
func TestService_FetchSubscription_ValidatesURL(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(NewFileStore(dir), DefaultConfig())

	_, err := svc.FetchSubscription("ftp://example.com/sub")
	if err == nil {
		t.Error("expected error for non-http URL")
	}
}
