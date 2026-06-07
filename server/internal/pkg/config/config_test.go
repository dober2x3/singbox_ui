package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_fileNotFound(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
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

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.ListenAddr != "0.0.0.0:8080" {
		t.Errorf("expected 0.0.0.0:8080, got %s", cfg.Server.ListenAddr)
	}
	if cfg.Prober.Interval != 30 {
		t.Errorf("expected default prober interval 30, got %d", cfg.Prober.Interval)
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

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Prober.Interval != 60 {
		t.Errorf("expected prober interval 60, got %d", cfg.Prober.Interval)
	}
	if cfg.Prober.Timeout != 3000 {
		t.Errorf("expected prober timeout 3000, got %d", cfg.Prober.Timeout)
	}
	if !cfg.Server.ServeDashboard {
		t.Error("expected serve_dashboard true")
	}
	if cfg.Speedtest.LatencyURL == "" {
		t.Error("expected speedtest defaults")
	}
}

func TestInit_withConfigFlag(t *testing.T) {
	os.Clearenv()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("server:\n  listen_addr: \"0.0.0.0:9090\"\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Init(path)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.Server.ListenAddr != "0.0.0.0:9090" {
		t.Errorf("expected 0.0.0.0:9090, got %s", cfg.Server.ListenAddr)
	}
}

func TestInit_defaultPath(t *testing.T) {
	os.Clearenv()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

	cfg, err := Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.GetSingboxDir() != filepath.Join(tmpDir, "singbox") {
		t.Errorf("GetSingboxDir() = %q, want %q", cfg.GetSingboxDir(), filepath.Join(tmpDir, "singbox"))
	}
}

func TestInit_withDataDirEnv(t *testing.T) {
	os.Clearenv()
	tmpDir := t.TempDir()
	os.Setenv("DATA_DIR", tmpDir)
	defer os.Unsetenv("DATA_DIR")

	cfg, err := Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.GetDataDir() != tmpDir {
		t.Errorf("GetDataDir() = %q, want %q", cfg.GetDataDir(), tmpDir)
	}
}

func TestResolveHostConfigDir(t *testing.T) {
	os.Clearenv()
	os.Setenv("DATA_DIR", "/home/data")
	os.Setenv("HOST_DATA_DIR", "/host/data")
	defer os.Unsetenv("DATA_DIR")
	defer os.Unsetenv("HOST_DATA_DIR")

	cfg, err := Init("")
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
	os.Clearenv()
	os.Setenv("DATA_DIR", "/home/data")
	defer os.Unsetenv("DATA_DIR")

	cfg, err := Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, err = cfg.ResolveHostConfigDir("/home/data/singbox")
	if err == nil {
		t.Error("ResolveHostConfigDir() expected error, got nil")
	}
}

func TestGetListenAddr_default(t *testing.T) {
	os.Clearenv()
	cfg, _ := Init("")
	if cfg.GetListenAddr() != "127.0.0.1:7000" {
		t.Errorf("GetListenAddr() = %q, want %q", cfg.GetListenAddr(), "127.0.0.1:7000")
	}
}

func TestGetListenAddr_custom(t *testing.T) {
	os.Clearenv()
	os.Setenv("LISTEN_ADDR", "0.0.0.0:8080")
	defer os.Unsetenv("LISTEN_ADDR")

	cfg, _ := Init("")
	if cfg.GetListenAddr() != "0.0.0.0:8080" {
		t.Errorf("GetListenAddr() = %q, want %q", cfg.GetListenAddr(), "0.0.0.0:8080")
	}
}
