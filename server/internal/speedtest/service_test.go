package speedtest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

// mockContainerAPI implements ContainerManager for testing.
type mockContainerAPI struct {
	ContainerCreateFn func(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error)
	ContainerStartFn  func(ctx context.Context, containerID string) error
	ContainerStopFn   func(ctx context.Context, containerID string, timeout *int) error
	ContainerRemoveFn func(ctx context.Context, containerID string, force bool) error
}

// newMockContainerAPI creates a mockContainerAPI with default successful responses.
func newMockContainerAPI() *mockContainerAPI {
	return &mockContainerAPI{
		ContainerCreateFn: func(_ context.Context, config interface{}, hostConfig interface{}, name string) (string, error) {
			return "mock-container-id", nil
		},
		ContainerStartFn: func(_ context.Context, containerID string) error {
			return nil
		},
		ContainerStopFn: func(_ context.Context, containerID string, timeout *int) error {
			return nil
		},
		ContainerRemoveFn: func(_ context.Context, containerID string, force bool) error {
			return nil
		},
	}
}

// ContainerCreate delegates to the mock function.
func (m *mockContainerAPI) ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error) {
	return m.ContainerCreateFn(ctx, config, hostConfig, name)
}

// ContainerStart delegates to the mock function.
func (m *mockContainerAPI) ContainerStart(ctx context.Context, containerID string) error {
	return m.ContainerStartFn(ctx, containerID)
}

// ContainerStop delegates to the mock function.
func (m *mockContainerAPI) ContainerStop(ctx context.Context, containerID string, timeout *int) error {
	return m.ContainerStopFn(ctx, containerID, timeout)
}

// ContainerRemove delegates to the mock function.
func (m *mockContainerAPI) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	return m.ContainerRemoveFn(ctx, containerID, force)
}

// mockNodeProvider implements NodeProvider for testing.
type mockNodeProvider struct {
	nodes []types.ProxyNode
}

// GetAllNodes returns the mock list of nodes.
func (m *mockNodeProvider) GetAllNodes() []types.ProxyNode {
	return m.nodes
}

// mockResultSaver implements SpeedTestResultSaver for testing.
type mockResultSaver struct {
	mu    sync.Mutex
	saved []types.SpeedTestUpdate
}

// SaveSpeedTestResults stores the results in memory.
func (m *mockResultSaver) SaveSpeedTestResults(results []types.SpeedTestUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, results...)
	return nil
}

// GetSaved returns a copy of all saved results.
func (m *mockResultSaver) GetSaved() []types.SpeedTestUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]types.SpeedTestUpdate, len(m.saved))
	copy(result, m.saved)
	return result
}

// TestPickFreePort verifies that pickFreePort returns a valid port number.
func TestPickFreePort(t *testing.T) {
	port, err := pickFreePort()
	if err != nil {
		t.Fatalf("pickFreePort() error = %v", err)
	}
	if port <= 0 {
		t.Errorf("port = %d, want > 0", port)
	}
}

// TestNewService verifies that NewService returns a non-nil Service with an initialized state.
func TestNewService(t *testing.T) {
	mockDocker := newMockContainerAPI()
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
	if svc.state == nil {
		t.Error("state should be initialized")
	}
}

// TestService_StartSpeedTest_NoProvider verifies that StartSpeedTest fails without a node provider.
func TestService_StartSpeedTest_NoProvider(t *testing.T) {
	mockDocker := newMockContainerAPI()
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)

	err = svc.StartSpeedTest()
	if err == nil {
		t.Fatal("expected error for no node provider")
	}
	if err.Error() != "node provider not configured" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestService_StartSpeedTest_NoNodes verifies that StartSpeedTest fails with an empty node list.
func TestService_StartSpeedTest_NoNodes(t *testing.T) {
	mockDocker := newMockContainerAPI()
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)
	svc.WithNodeProvider(&mockNodeProvider{nodes: []types.ProxyNode{}})

	err = svc.StartSpeedTest()
	if err == nil {
		t.Fatal("expected error for no nodes")
	}
	if err.Error() != "no nodes to test" {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestService_StartSpeedTest_AlreadyRunning verifies that starting a second test returns an error.
func TestService_StartSpeedTest_AlreadyRunning(t *testing.T) {
	mockDocker := newMockContainerAPI()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)
	svc.nodeProvider = &mockNodeProvider{
		nodes: []types.ProxyNode{
			{Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443}},
		},
	}

	err = svc.StartSpeedTest()
	if err != nil {
		t.Fatalf("first StartSpeedTest() error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	err = svc.StartSpeedTest()
	if err == nil {
		t.Fatal("expected error for already running")
	}
}

// TestService_GetSpeedTestState verifies the initial state is not running.
func TestService_GetSpeedTestState(t *testing.T) {
	mockDocker := newMockContainerAPI()
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)

	state := svc.GetSpeedTestState()
	if state == nil {
		t.Fatal("GetSpeedTestState() returned nil")
	}
	if state.Running {
		t.Error("state should not be running initially")
	}
}

// TestService_StopSpeedTest verifies that stopping a running test sets state to not running.
func TestService_StopSpeedTest(t *testing.T) {
	mockDocker := newMockContainerAPI()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)
	svc.nodeProvider = &mockNodeProvider{
		nodes: []types.ProxyNode{
			{Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443}},
		},
	}

	err = svc.StartSpeedTest()
	if err != nil {
		t.Fatalf("StartSpeedTest() error = %v", err)
	}

	svc.StopSpeedTest()

	time.Sleep(100 * time.Millisecond)

	state := svc.GetSpeedTestState()
	if state.Running {
		t.Error("state should not be running after stop")
	}
}

// TestService_RunSpeedTest_ContainerStartFails verifies cleanup when container start fails.
func TestService_RunSpeedTest_ContainerStartFails(t *testing.T) {
	mockDocker := newMockContainerAPI()
	var created, started, removed atomic.Bool
	mockDocker.ContainerCreateFn = func(_ context.Context, config, hostConfig interface{}, name string) (string, error) {
		created.Store(true)
		return "speedtest-id", nil
	}
	mockDocker.ContainerStartFn = func(_ context.Context, id string) error {
		started.Store(true)
		return fmt.Errorf("simulated start failure")
	}
	mockDocker.ContainerRemoveFn = func(_ context.Context, id string, force bool) error {
		removed.Store(true)
		return nil
	}

	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(mockDocker, cfg)
	svc.WithNodeProvider(&mockNodeProvider{
		nodes: []types.ProxyNode{
			{Name: "test-node", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443}},
		},
	})

	if err := svc.StartSpeedTest(); err != nil {
		t.Fatalf("StartSpeedTest() error = %v", err)
	}

	time.Sleep(2 * time.Second)

	state := svc.GetSpeedTestState()
	if state.Running {
		svc.StopSpeedTest()
		t.Error("speed test should have completed")
	}

	if !created.Load() {
		t.Error("ContainerCreate should have been called")
	}
	if !started.Load() {
		t.Error("ContainerStart should have been called")
	}
	if !removed.Load() {
		t.Error("ContainerRemove should have been called on failure")
	}
}

// TestService_RunSpeedTest_WithResultSaver verifies that results are saved when a result saver is set.
func TestService_RunSpeedTest_WithResultSaver(t *testing.T) {
	mockDocker := newMockContainerAPI()
	mockDocker.ContainerCreateFn = func(_ context.Context, config, hostConfig interface{}, name string) (string, error) {
		return "speedtest-id", nil
	}
	mockDocker.ContainerStartFn = func(_ context.Context, id string) error {
		return fmt.Errorf("simulated start failure")
	}
	mockDocker.ContainerRemoveFn = func(_ context.Context, id string, force bool) error {
		return nil
	}

	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}

	saver := &mockResultSaver{}
	svc := NewService(mockDocker, cfg)
	svc.WithNodeProvider(&mockNodeProvider{
		nodes: []types.ProxyNode{
			{Name: "test-node", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443}},
		},
	})
	svc.WithResultSaver(saver)

	if err := svc.StartSpeedTest(); err != nil {
		t.Fatalf("StartSpeedTest() error = %v", err)
	}

	time.Sleep(2 * time.Second)

	if len(saver.GetSaved()) == 0 {
		t.Error("result saver should have been called")
	}
}

// TestBuildSpeedTestConfig verifies the generated sing-box config structure.
func TestBuildSpeedTestConfig(t *testing.T) {
	node := &types.ProxyNode{
		Name:     "test-node",
		Protocol: "vmess",
		Address:  "1.1.1.1",
		Port:     443,
		Outbound: map[string]interface{}{
			"type":        "vmess",
			"server":      "1.1.1.1",
			"server_port": 443,
		},
	}

	cfg := buildSpeedTestConfig(node, "vmess-1_1_1_1-443", 10800)
	if cfg == nil {
		t.Fatal("buildSpeedTestConfig() returned nil")
	}

	outbounds, ok := cfg["outbounds"].([]map[string]interface{})
	if !ok {
		t.Fatal("outbounds should be []map[string]interface{}")
	}
	if len(outbounds) != 1 {
		t.Errorf("expected 1 outbound, got %d", len(outbounds))
	}

	ob := outbounds[0]
	if ob["tag"] != "vmess-1_1_1_1-443" {
		t.Errorf("outbound tag = %v, want vmess-1_1_1_1-443", ob["tag"])
	}

	inbounds, ok := cfg["inbounds"].([]map[string]interface{})
	if !ok {
		t.Fatal("inbounds should be []map[string]interface{}")
	}
	if len(inbounds) != 1 {
		t.Errorf("expected 1 inbound, got %d", len(inbounds))
	}

	ib := inbounds[0]
	if ib["listen_port"] != 10800 {
		t.Errorf("listen_port = %v, want 10800", ib["listen_port"])
	}
}

// TestNodeOutboundTag verifies tag resolution from outbound config and fallback generation.
func TestNodeOutboundTag(t *testing.T) {
	node := &types.ProxyNode{
		Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
		Outbound: map[string]interface{}{"tag": "custom-tag"},
	}
	if tag := nodeOutboundTag(node); tag != "custom-tag" {
		t.Errorf("nodeOutboundTag() = %s, want custom-tag", tag)
	}

	node.Outbound = nil
	if tag := nodeOutboundTag(node); tag != "vmess-1_1_1_1-443" {
		t.Errorf("nodeOutboundTag() = %s, want vmess-1_1_1_1-443", tag)
	}
}

// TestNewProxyClient verifies the proxy http.Client is created with the correct timeout.
func TestNewProxyClient(t *testing.T) {
	client := newProxyClient("http://127.0.0.1:1080", 10*time.Second)
	if client == nil {
		t.Fatal("newProxyClient() returned nil")
	}
	if client.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", client.Timeout)
	}
}
