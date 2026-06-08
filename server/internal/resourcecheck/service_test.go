package resourcecheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

type mockNodeProvider struct {
	nodes []types.ProxyNode
}

func (m *mockNodeProvider) GetAllNodes() ([]types.ProxyNode, error) {
	return m.nodes, nil
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)

	cfg, err := config.Init("")
	if err != nil {
		t.Fatal(err)
	}

	testDataDir := filepath.Join(cfgDir, "testdata")
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	resourcesPath := filepath.Join(testDataDir, "resources.yaml")
	if err := os.WriteFile(resourcesPath, []byte(`
resources:
  - name: youtube
    url: https://www.youtube.com
    type: http
  - name: telegram
    url: https://telegram.org
    type: http
`), 0644); err != nil {
		t.Fatal(err)
	}

	checker := NewChecker(&mockRunner{}, cfg)
	np := &mockNodeProvider{
		nodes: []types.ProxyNode{
			{
				Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
			},
		},
	}

	dbPath := filepath.Join(cfgDir, "test.db")
	svc := NewService(checker, np, ProberConfig{
		ResourcesPath: resourcesPath,
		DBPath:        dbPath,
	})
	return svc
}

func TestService_Init(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer svc.Close()

	resources := svc.GetResources()
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
}

func TestService_RunAll_NoNodes(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, _ := config.Init("")
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: filepath.Join(cfgDir, "test.db"),
	})

	err := svc.RunAll(context.Background())
	if err == nil {
		t.Fatal("expected error for no nodes")
	}
}

func TestService_RunForTag_NotFound(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, _ := config.Init("")
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: filepath.Join(cfgDir, "test.db"),
	})

	err := svc.RunForTag(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
}

func TestService_GetStatus_Initial(t *testing.T) {
	cfgDir := t.TempDir()
	cfg, _ := config.Init("")
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: filepath.Join(cfgDir, "test.db"),
	})

	status := svc.GetStatus()
	if status.Status != "idle" {
		t.Errorf("expected idle, got %s", status.Status)
	}
}

func TestService_Stop_Idempotent(t *testing.T) {
	cfgDir := t.TempDir()
	cfg, _ := config.Init("")
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: filepath.Join(cfgDir, "test.db"),
	})

	svc.Stop()
	svc.Stop()
}

func TestService_ReloadResources(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer svc.Close()

	if len(svc.GetResources()) != 2 {
		t.Fatalf("expected 2 resources initially")
	}

	if err := svc.ReloadResources(); err != nil {
		t.Fatalf("ReloadResources() error = %v", err)
	}
	if len(svc.GetResources()) != 2 {
		t.Errorf("expected 2 resources after reload")
	}
}

func TestService_RunAll_WithResources(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer svc.Close()

	err := svc.RunAll(context.Background())
	if err != nil {
		t.Logf("RunAll returned error (expected with mock): %v", err)
	}
}

func TestService_Scheduler_StartStop(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	defer svc.Close()

	svc.StartScheduler(3600)
	svc.StopScheduler()
}

func TestService_Scheduler_NegativeInterval(t *testing.T) {
	svc := newTestService(t)
	svc.StartScheduler(-1)
}

func TestService_GetHistory_StoreNil(t *testing.T) {
	svc := NewService(nil, &mockNodeProvider{}, ProberConfig{})
	results, err := svc.GetHistory("youtube", "test", 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if results != nil {
		t.Errorf("expected nil, got %v", results)
	}
}
