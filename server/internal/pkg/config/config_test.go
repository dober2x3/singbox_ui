package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/prober"
	"singbox-config-service/internal/speedtest"

	"gopkg.in/yaml.v3"
)

func TestLoadConfig_fileNotFound(t *testing.T) {
	cfg, err := config.Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("Load() with missing file should not error: %v", err)
	}
	if cfg.Server.ListenAddr != "127.0.0.1:7000" {
		t.Errorf("expected default listen addr, got %s", cfg.Server.ListenAddr)
	}
}

func TestLoadConfig_partial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("server:\n  listen_addr: \"0.0.0.0:8080\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.ListenAddr != "0.0.0.0:8080" {
		t.Errorf("expected 0.0.0.0:8080, got %s", cfg.Server.ListenAddr)
	}
	// Verify domain sections parse with defaults
	pc, err := prober.ParseConfig(&cfg.Prober)
	if err != nil {
		t.Fatalf("ParseConfig(prober) error = %v", err)
	}
	if pc.Interval != 30 {
		t.Errorf("expected default prober interval 30, got %d", pc.Interval)
	}
}

func TestLoadConfig_full(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
server:
  listen_addr: "0.0.0.0:7000"
  data_dir: "/data"
  serve_dashboard: true
prober:
  interval: 60
  timeout: 3000
  concurrent: 10
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Server.ServeDashboard {
		t.Error("expected serve_dashboard true")
	}

	// Check prober fields via ParseConfig
	pc, err := prober.ParseConfig(&cfg.Prober)
	if err != nil {
		t.Fatalf("ParseConfig(prober) error = %v", err)
	}
	if pc.Interval != 60 {
		t.Errorf("expected prober interval 60, got %d", pc.Interval)
	}
	if pc.Timeout != 3000 {
		t.Errorf("expected prober timeout 3000, got %d", pc.Timeout)
	}
	if pc.Concurrent != 10 {
		t.Errorf("expected prober concurrent 10, got %d", pc.Concurrent)
	}

	// Check speedtest defaults via ParseConfig
	sc, err := speedtest.ParseConfig(&cfg.Speedtest)
	if err != nil {
		t.Fatalf("ParseConfig(speedtest) error = %v", err)
	}
	if sc.LatencyURL == "" {
		t.Error("expected speedtest defaults")
	}
}

func TestInit_withConfigFlag(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("HOST_DATA_DIR", "")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("server:\n  listen_addr: \"0.0.0.0:9090\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Init(path)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.Server.ListenAddr != "0.0.0.0:9090" {
		t.Errorf("expected 0.0.0.0:9090, got %s", cfg.Server.ListenAddr)
	}
}

func TestInit_defaultDataDir(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("HOST_DATA_DIR", "")

	cfg, err := config.Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !strings.HasSuffix(cfg.GetDataDir(), "singbox_ui") {
		t.Errorf("expected GetDataDir() to end with 'singbox_ui', got %q", cfg.GetDataDir())
	}
}

func TestInit_withDataDirEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DATA_DIR", tmpDir)
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("HOST_DATA_DIR", "")

	cfg, err := config.Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.GetDataDir() != tmpDir {
		t.Errorf("GetDataDir() = %q, want %q", cfg.GetDataDir(), tmpDir)
	}
}

func TestResolveHostConfigDir(t *testing.T) {
	t.Setenv("DATA_DIR", "/home/data")
	t.Setenv("HOST_DATA_DIR", "/host/data")
	defer os.Unsetenv("DATA_DIR")
	defer os.Unsetenv("HOST_DATA_DIR")

	cfg, err := config.Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	hostPath, err := cfg.ResolveHostConfigDir("/home/data/singbox")
	if err != nil {
		t.Fatalf("ResolveHostConfigDir() error = %v", err)
	}
	want := "/host/data/singbox"
	if hostPath != want {
		t.Errorf("ResolveHostConfigDir() = %q, want %q", hostPath, want)
	}
}

func TestResolveHostConfigDir_noHostDir(t *testing.T) {
	t.Setenv("DATA_DIR", "/home/data")
	t.Setenv("HOST_DATA_DIR", "")
	defer os.Unsetenv("DATA_DIR")

	cfg, err := config.Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, err = cfg.ResolveHostConfigDir("/home/data/singbox")
	if err == nil {
		t.Error("ResolveHostConfigDir() expected error, got nil")
	}
}

func TestGetListenAddr_default(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("HOST_DATA_DIR", "")
	cfg, _ := config.Init("")
	if cfg.GetListenAddr() != "127.0.0.1:7000" {
		t.Errorf("GetListenAddr() = %q, want %q", cfg.GetListenAddr(), "127.0.0.1:7000")
	}
}

func TestGetListenAddr_custom(t *testing.T) {
	t.Setenv("LISTEN_ADDR", "0.0.0.0:8080")
	t.Setenv("DATA_DIR", "")
	t.Setenv("HOST_DATA_DIR", "")
	defer os.Unsetenv("LISTEN_ADDR")

	cfg, _ := config.Init("")
	if cfg.GetListenAddr() != "0.0.0.0:8080" {
		t.Errorf("GetListenAddr() = %q, want %q", cfg.GetListenAddr(), "0.0.0.0:8080")
	}
}

func TestYamlNodeRoundTrip(t *testing.T) {
	// Verify that yaml.Node correctly captures and decodes a config section
	content := []byte(`
prober:
  interval: 45
  timeout: 2000
`)
	var cfg config.AppConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		t.Fatal(err)
	}

	pc, err := prober.ParseConfig(&cfg.Prober)
	if err != nil {
		t.Fatal(err)
	}
	if pc.Interval != 45 {
		t.Errorf("expected interval 45, got %d", pc.Interval)
	}
	if pc.Timeout != 2000 {
		t.Errorf("expected timeout 2000, got %d", pc.Timeout)
	}
}
