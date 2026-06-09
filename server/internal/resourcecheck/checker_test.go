package resourcecheck

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

// mockRunner implements Runner for testing.
type mockRunner struct {
	mu          sync.Mutex
	startCalled bool
	stopCalled  bool
	startErr    error
	stopErr     error
	readyErr    error
	instanceID  string
}

func (m *mockRunner) StartTemp(ctx context.Context, configPath string) (string, error) {
	m.mu.Lock()
	m.startCalled = true
	id := m.instanceID
	err := m.startErr
	m.mu.Unlock()
	if id == "" {
		id = "mock-id"
	}
	return id, err
}

func (m *mockRunner) StopTemp(ctx context.Context, id string) error {
	m.mu.Lock()
	m.stopCalled = true
	err := m.stopErr
	m.mu.Unlock()
	return err
}

func (m *mockRunner) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	return m.readyErr
}

func (m *mockRunner) GetTempLogs(ctx context.Context, id string) string {
	return ""
}

func TestChecker_CheckNodeResources_MissingOutbound(t *testing.T) {
	cfg, _ := config.Init("")
	checker := NewChecker(&mockRunner{}, cfg)
	node := &types.ProxyNode{Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443}

	_, err := checker.CheckNodeResources(context.Background(), node, nil)
	if err == nil {
		t.Fatal("expected error for missing outbound")
	}
}

func TestChecker_CheckNodeResources_TunnelStartFails(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init("")
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockRunner{startErr: fmt.Errorf("start failed")}
	checker := NewChecker(mock, cfg)

	node := &types.ProxyNode{
		Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
		Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
	}

	_, err = checker.CheckNodeResources(context.Background(), node, []ResourceConfig{
		{Name: "youtube", URL: "https://www.youtube.com", Type: "http"},
	})
	if err == nil {
		t.Fatal("expected error when tunnel start fails")
	}
}

func TestNodeOutboundTag(t *testing.T) {
	node := &types.ProxyNode{
		Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
		Outbound: map[string]interface{}{"tag": "custom-tag"},
	}
	if tag := nodeOutboundTag(node); tag != "custom-tag" {
		t.Errorf("nodeOutboundTag() = %s, want custom-tag", tag)
	}
	node.Outbound = nil
	if tag := nodeOutboundTag(node); tag != "vmess-1.1.1.1-443" {
		t.Errorf("nodeOutboundTag() = %s, want vmess-1.1.1.1-443", tag)
	}
}

func TestBuildCheckConfig(t *testing.T) {
	node := &types.ProxyNode{
		Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
		Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
	}
	cfg := buildCheckConfig(node, "vmess-1_1_1_1-443", 10800)
	if cfg == nil {
		t.Fatal("buildCheckConfig returned nil")
	}
}

func TestPickFreePort(t *testing.T) {
	port, err := pickFreePort()
	if err != nil {
		t.Fatalf("pickFreePort() error = %v", err)
	}
	if port <= 0 {
		t.Errorf("port = %d, want > 0", port)
	}
}
