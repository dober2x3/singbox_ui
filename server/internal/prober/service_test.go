package prober

import (
	"testing"

	"singbox-config-service/internal/pkg/types"
)

// mockResultSaver is a no-op implementation of ProbeResultSaver for testing.
type mockResultSaver struct{}

// SaveProbeResults is a no-op implementation for testing.
func (m *mockResultSaver) SaveProbeResults(results []types.ProbeResultUpdate) error {
	return nil
}

// TestNewService verifies that NewService creates a non-nil service.
func TestNewService(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

// TestServiceInit verifies that Init creates the underlying prober.
func TestServiceInit(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if svc.prober == nil {
		t.Error("prober should be initialized after Init()")
	}
}

// TestServiceStartStop verifies the service start/stop lifecycle.
func TestServiceStartStop(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	svc.Start()
	if !svc.IsRunning() {
		t.Error("Service should be running after Start()")
	}

	svc.Stop()
	if svc.IsRunning() {
		t.Error("Service should not be running after Stop()")
	}
}

// TestServiceCRUD verifies basic add/get/remove operations through the service.
func TestServiceCRUD(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	svc.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})

	result := svc.GetResult("test")
	if result == nil {
		t.Fatal("GetResult() returned nil after AddNode()")
	}

	svc.RemoveNode("test")
	if svc.GetResult("test") != nil {
		t.Error("GetResult() should return nil after RemoveNode()")
	}
}

// TestServiceSaveProbeResults verifies saving probe results via the result saver.
func TestServiceSaveProbeResults(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	svc.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443})

	count, err := svc.SaveProbeResults()
	if err != nil {
		t.Fatalf("SaveProbeResults() error = %v", err)
	}
	if count != 1 {
		t.Errorf("SaveProbeResults() = %d, want 1", count)
	}
}

// TestServiceSaveProbeResults_Empty verifies saving with no results returns zero.
func TestServiceSaveProbeResults_Empty(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	count, err := svc.SaveProbeResults()
	if err != nil {
		t.Fatalf("SaveProbeResults() error = %v", err)
	}
	if count != 0 {
		t.Errorf("SaveProbeResults() = %d, want 0", count)
	}
}

// TestServiceSaveNodes verifies saving and loading nodes through the service.
func TestServiceSaveNodes(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	svc.AddNode(types.ProbeNode{Tag: "saved-node", Protocol: "vmess", Address: "1.1.1.1", Port: 443})

	if err := svc.SaveNodes(); err != nil {
		t.Fatalf("SaveNodes() error = %v", err)
	}

	svc.ClearNodes()
	if len(svc.GetAllResults()) != 0 {
		t.Error("ClearNodes() should clear all nodes")
	}

	if err := svc.LoadNodes(); err != nil {
		t.Fatalf("LoadNodes() error = %v", err)
	}

	if len(svc.GetAllResults()) != 1 {
		t.Errorf("Expected 1 node after load, got %d", len(svc.GetAllResults()))
	}
}

// TestServiceGetAllResults verifies GetAllResults returns an empty map initially.
func TestServiceGetAllResults(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(DefaultConfig(), dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	results := svc.GetAllResults()
	if results == nil {
		t.Error("GetAllResults() should return empty map, not nil")
	}
	if len(results) != 0 {
		t.Errorf("GetAllResults() should be empty, got %d", len(results))
	}
}
