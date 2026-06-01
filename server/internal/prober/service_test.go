package prober

import (
	"testing"

	"singbox-config-service/internal/pkg/types"
)

type mockResultSaver struct{}

func (m *mockResultSaver) SaveProbeResults(results []types.ProbeResultUpdate) error {
	return nil
}

func TestNewService(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestServiceInit(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if svc.prober == nil {
		t.Error("prober should be initialized after Init()")
	}
}

func TestServiceStartStop(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)

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

func TestServiceCRUD(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)

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

func TestServiceSaveProbeResults(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)

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

func TestServiceSaveProbeResults_Empty(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)

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

func TestServiceSaveNodes(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)

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

func TestServiceGetAllResults(t *testing.T) {
	dir := t.TempDir()
	saver := &mockResultSaver{}
	svc := NewService(dir, saver)

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
