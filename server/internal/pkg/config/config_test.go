package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit_defaultPath(t *testing.T) {
	os.Clearenv()
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	// Create go.mod to simulate server directory
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

	cfg, err := Init()
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

	cfg, err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.GetSingboxDir() != filepath.Join(tmpDir, "singbox") {
		t.Errorf("GetSingboxDir() = %q, want %q", cfg.GetSingboxDir(), filepath.Join(tmpDir, "singbox"))
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

	cfg, err := Init()
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

	cfg, err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, err = cfg.ResolveHostConfigDir("/home/data/singbox")
	if err == nil {
		t.Error("ResolveHostConfigDir() expected error, got nil")
	}
}

func TestResolveHostConfigDir_outsideDataDir(t *testing.T) {
	os.Clearenv()
	os.Setenv("DATA_DIR", "/home/data")
	os.Setenv("HOST_DATA_DIR", "/host/data")
	defer os.Unsetenv("DATA_DIR")
	defer os.Unsetenv("HOST_DATA_DIR")

	cfg, err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	_, err = cfg.ResolveHostConfigDir("/outside/data")
	if err == nil {
		t.Error("ResolveHostConfigDir() expected error for path outside DATA_DIR, got nil")
	}
}

func TestGetListenAddr_default(t *testing.T) {
	os.Clearenv()
	cfg, _ := Init()
	if cfg.GetListenAddr() != "127.0.0.1:7000" {
		t.Errorf("GetListenAddr() = %q, want %q", cfg.GetListenAddr(), "127.0.0.1:7000")
	}
}

func TestGetListenAddr_custom(t *testing.T) {
	os.Clearenv()
	os.Setenv("LISTEN_ADDR", "0.0.0.0:8080")
	defer os.Unsetenv("LISTEN_ADDR")
	cfg, _ := Init()
	if cfg.GetListenAddr() != "0.0.0.0:8080" {
		t.Errorf("GetListenAddr() = %q, want %q", cfg.GetListenAddr(), "0.0.0.0:8080")
	}
}
