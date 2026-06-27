# Config Dependency Inversion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Invert config dependency direction so `pkg/config` no longer imports domain packages, and change default DataDir to `$HOME/singbox_ui`.

**Architecture:** `pkg/config` stores domain-specific config sections as `yaml.Node` fields. Each domain package (prober, scheduler, subscription, speedtest) provides `ParseConfig(*yaml.Node) (Config, error)` to parse its own section. `main.go` uses ParseConfig instead of direct field access.

**Tech Stack:** Go, gopkg.in/yaml.v3, os/user

---

### Task 1: Add ParseConfig to prober/models.go

**Files:**
- Modify: `server/internal/prober/models.go`

- [ ] **Step 1: Read current file**

Read `server/internal/prober/models.go` to check structure.

- [ ] **Step 2: Add yaml.v3 import and ParseConfig function**

Edit `server/internal/prober/models.go`:

Add import block:
```go
import (
	"gopkg.in/yaml.v3"
)
```

Append after `DefaultConfig()`:
```go
// ParseConfig parses a yaml.Node into a Config, applying defaults.
// Returns DefaultConfig if node is nil or zero-valued.
func ParseConfig(node *yaml.Node) (Config, error) {
	cfg := DefaultConfig()
	if node == nil || node.Kind == 0 {
		return cfg, nil
	}
	if err := node.Decode(&cfg); err != nil {
		return Config{}, err
	}
	def := DefaultConfig()
	if cfg.Interval == 0 {
		cfg.Interval = def.Interval
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = def.Timeout
	}
	if cfg.Concurrent == 0 {
		cfg.Concurrent = def.Concurrent
	}
	if cfg.MaxResults == 0 {
		cfg.MaxResults = def.MaxResults
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = def.MaxRetries
	}
	return cfg, nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/kev/work/projects/singbox_ui/server && go build ./internal/prober/`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add server/internal/prober/models.go
git commit -m "refactor(prober): add ParseConfig for self-contained config parsing"
```

---

### Task 2: Add ParseConfig to scheduler/models.go

**Files:**
- Modify: `server/internal/scheduler/models.go`

- [ ] **Step 1: Read current file**

Read `server/internal/scheduler/models.go`.

- [ ] **Step 2: Add yaml.v3 import and ParseConfig function**

Edit `server/internal/scheduler/models.go`:

Add import block:
```go
import (
	"gopkg.in/yaml.v3"
)
```

Append after `DefaultConfig()`:
```go
// ParseConfig parses a yaml.Node into a Config, applying defaults.
func ParseConfig(node *yaml.Node) (Config, error) {
	cfg := DefaultConfig()
	if node == nil || node.Kind == 0 {
		return cfg, nil
	}
	if err := node.Decode(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.Interval == 0 {
		cfg.Interval = DefaultConfig().Interval
	}
	return cfg, nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/kev/work/projects/singbox_ui/server && go build ./internal/scheduler/`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add server/internal/scheduler/models.go
git commit -m "refactor(scheduler): add ParseConfig for self-contained config parsing"
```

---

### Task 3: Add ParseConfig to subscription/models.go

**Files:**
- Modify: `server/internal/subscription/models.go`

- [ ] **Step 1: Read current file**

Read `server/internal/subscription/models.go`.

- [ ] **Step 2: Add yaml.v3 import and ParseConfig function**

Edit `server/internal/subscription/models.go`:

Add import block:
```go
import (
	"gopkg.in/yaml.v3"
)
```

Append after `DefaultConfig()`:
```go
// ParseConfig parses a yaml.Node into a Config, applying defaults.
func ParseConfig(node *yaml.Node) (Config, error) {
	cfg := DefaultConfig()
	if node == nil || node.Kind == 0 {
		return cfg, nil
	}
	if err := node.Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/kev/work/projects/singbox_ui/server && go build ./internal/subscription/`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add server/internal/subscription/models.go
git commit -m "refactor(subscription): add ParseConfig for self-contained config parsing"
```

---

### Task 4: Add ParseConfig to speedtest/models.go

**Files:**
- Modify: `server/internal/speedtest/models.go`

- [ ] **Step 1: Read current file**

Read `server/internal/speedtest/models.go`.

- [ ] **Step 2: Add yaml.v3 import and ParseConfig function**

Edit `server/internal/speedtest/models.go`:

Add import block:
```go
import (
	"gopkg.in/yaml.v3"
)
```

Append after `DefaultConfig()`:
```go
// ParseConfig parses a yaml.Node into a Config, applying defaults.
func ParseConfig(node *yaml.Node) (Config, error) {
	cfg := DefaultConfig()
	if node == nil || node.Kind == 0 {
		return cfg, nil
	}
	if err := node.Decode(&cfg); err != nil {
		return Config{}, err
	}
	def := DefaultConfig()
	if cfg.LatencyURL == "" {
		cfg.LatencyURL = def.LatencyURL
	}
	if cfg.DownloadURL == "" {
		cfg.DownloadURL = def.DownloadURL
	}
	if cfg.Duration == 0 {
		cfg.Duration = def.Duration
	}
	return cfg, nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/kev/work/projects/singbox_ui/server && go build ./internal/speedtest/`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add server/internal/speedtest/models.go
git commit -m "refactor(speedtest): add ParseConfig for self-contained config parsing"
```

---

### Task 5: Refactor pkg/config/config.go

**Files:**
- Modify: `server/internal/pkg/config/config.go`

- [ ] **Step 1: Read current file**

Read `server/internal/pkg/config/config.go` to confirm current state.

- [ ] **Step 2: Rewrite config.go**

Replace the entire file content with:

```go
// Package config provides application configuration loaded from environment variables
// and filesystem state. It manages data directories, listen addresses, and host path
// resolution for container environments.
package config

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds HTTP server and path configuration.
type ServerConfig struct {
	ListenAddr      string `yaml:"listen_addr"`
	DataDir         string `yaml:"data_dir"`
	HostDataDir     string `yaml:"host_data_dir"`
	SingboxBinPath  string `yaml:"singbox_bin_path"`
	ServeDashboard  bool   `yaml:"serve_dashboard"`
}

// AppConfig is the top-level application configuration.
// Domain-specific sections (Prober, Speedtest, Scheduler, Subscription) are stored
// as raw yaml.Node and parsed by each domain's ParseConfig function.
type AppConfig struct {
	Server       ServerConfig `yaml:"server"`
	Prober       yaml.Node    `yaml:"prober"`
	Speedtest    yaml.Node    `yaml:"speedtest"`
	Scheduler    yaml.Node    `yaml:"scheduler"`
	Subscription yaml.Node    `yaml:"subscription"`
}

// defaultDataDir returns the default data directory: $HOME/singbox_ui.
// Falls back to working directory if home dir cannot be determined.
func defaultDataDir() string {
	if u, err := user.Current(); err == nil {
		return filepath.Join(u.HomeDir, "singbox_ui")
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// Load reads a YAML config file and returns an AppConfig with defaults applied.
func Load(path string) (*AppConfig, error) {
	var cfg AppConfig
	cfg.Server.ListenAddr = "127.0.0.1:7000"

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if cfg.Server.ListenAddr == "" {
		cfg.Server.ListenAddr = "127.0.0.1:7000"
	}

	return &cfg, nil
}

// Init bootstraps configuration. It resolves the data directory and config file path.
// If configPath is non-empty, it is used directly.
// Otherwise DATA_DIR env var is used, or $HOME/singbox_ui as fallback.
func Init(configPath string) (*AppConfig, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = defaultDataDir()
	}

	path := configPath
	if path == "" {
		path = filepath.Join(dataDir, "config.yaml")
	}

	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	if cfg.Server.DataDir == "" {
		cfg.Server.DataDir = dataDir
	}

	// Backwards compatibility: if config file did not set listen_addr/host_data_dir,
	// check old env vars.
	if cfg.Server.ListenAddr == "127.0.0.1:7000" {
		if envAddr := os.Getenv("LISTEN_ADDR"); envAddr != "" {
			cfg.Server.ListenAddr = envAddr
		}
	}
	if cfg.Server.HostDataDir == "" {
		if envHost := os.Getenv("HOST_DATA_DIR"); envHost != "" {
			cfg.Server.HostDataDir = envHost
		}
	}

	singboxDir := filepath.Join(cfg.Server.DataDir, "singbox")
	if err := os.MkdirAll(singboxDir, 0755); err != nil {
		log.Printf("Warning: failed to create singbox directory: %v", err)
	}

	return cfg, nil
}

// GetDataDir returns the application data directory path.
func (c *AppConfig) GetDataDir() string {
	return c.Server.DataDir
}

// GetListenAddr returns the HTTP server listen address (host:port).
func (c *AppConfig) GetListenAddr() string {
	return c.Server.ListenAddr
}

// GetSingboxDir returns the sing-box configuration directory path.
func (c *AppConfig) GetSingboxDir() string {
	return filepath.Join(c.Server.DataDir, "singbox")
}

// GetSingboxBinPath returns the path to the sing-box binary for native mode.
func (c *AppConfig) GetSingboxBinPath() string {
	return c.Server.SingboxBinPath
}

// ResolveHostConfigDir converts a container-internal path under DATA_DIR to the
// corresponding host path using HOST_DATA_DIR.
func (c *AppConfig) ResolveHostConfigDir(containerPath string) (string, error) {
	if c.Server.HostDataDir == "" {
		return "", fmt.Errorf("HOST_DATA_DIR environment variable is not set")
	}
	rel, err := filepath.Rel(c.Server.DataDir, containerPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is not under DATA_DIR %s", containerPath, c.Server.DataDir)
	}
	return filepath.Join(c.Server.HostDataDir, rel), nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/kev/work/projects/singbox_ui/server && go build ./internal/pkg/config/`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add server/internal/pkg/config/config.go
git commit -m "refactor(config): invert dependencies, use yaml.Node for domain sections, default DataDir to $HOME/singbox_ui"
```

---

### Task 6: Update main.go to use ParseConfig

**Files:**
- Modify: `server/main.go`

- [ ] **Step 1: Read current wiring section**

Read `server/main.go` lines 88-99 (the service initialization section).

- [ ] **Step 2: Replace direct config access with ParseConfig calls**

In `server/main.go`, replace:

```go
	subSvc := subscription.NewService(subStore, cfg.Subscription)
	proberSvc := prober.NewService(cfg.Prober, cfg.GetDataDir(), subSvc)

	// Create shared tunnel runner
	tr := tunnelrunner.NewRunner(cfg)

	speedtestSvc := speedtest.NewService(tr, cfg, speedtest.Config{
		LatencyURL:  cfg.Speedtest.LatencyURL,
		DownloadURL: cfg.Speedtest.DownloadURL,
		Duration:    cfg.Speedtest.Duration,
	})
```

With:

```go
	subCfg, _ := subscription.ParseConfig(&cfg.Subscription)
	subSvc := subscription.NewService(subStore, subCfg)

	proberCfg, _ := prober.ParseConfig(&cfg.Prober)
	proberSvc := prober.NewService(proberCfg, cfg.GetDataDir(), subSvc)

	// Create shared tunnel runner
	tr := tunnelrunner.NewRunner(cfg)

	speedCfg, _ := speedtest.ParseConfig(&cfg.Speedtest)
	speedtestSvc := speedtest.NewService(tr, cfg, speedCfg)
```

Then replace:

```go
	sched := scheduler.New(subSvc, nil, cfg.Scheduler)
```

With:

```go
	schedCfg, _ := scheduler.ParseConfig(&cfg.Scheduler)
	sched := scheduler.New(subSvc, nil, schedCfg)
```

- [ ] **Step 3: Verify it compiles**

Run: `cd /home/kev/work/projects/singbox_ui/server && go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add server/main.go
git commit -m "refactor(main): use domain ParseConfig functions instead of direct config access"
```

---

### Task 7: Update config_test.go

**Files:**
- Modify: `server/internal/pkg/config/config_test.go`

- [ ] **Step 1: Read current test file**

Read `server/internal/pkg/config/config_test.go`.

- [ ] **Step 2: Rewrite the test file**

Replace the entire file with:

```go
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"singbox-config-service/internal/prober"
	"singbox-config-service/internal/speedtest"

	"gopkg.in/yaml.v3"
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

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check server fields
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

	cfg, err := Init(path)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.Server.ListenAddr != "0.0.0.0:9090" {
		t.Errorf("expected 0.0.0.0:9090, got %s", cfg.Server.ListenAddr)
	}
}

func TestInit_defaultDataDir(t *testing.T) {
	// defaultDataDir uses user.Current().HomeDir — verify the path ends with singbox_ui
	dataDir := defaultDataDir()
	if !strings.HasSuffix(dataDir, "singbox_ui") {
		t.Errorf("expected defaultDataDir to end with 'singbox_ui', got %q", dataDir)
	}
}

func TestInit_withDataDirEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DATA_DIR", tmpDir)
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("HOST_DATA_DIR", "")

	cfg, err := Init("")
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
	t.Setenv("DATA_DIR", "/home/data")
	t.Setenv("HOST_DATA_DIR", "")
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
	t.Setenv("DATA_DIR", "")
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("HOST_DATA_DIR", "")
	cfg, _ := Init("")
	if cfg.GetListenAddr() != "127.0.0.1:7000" {
		t.Errorf("GetListenAddr() = %q, want %q", cfg.GetListenAddr(), "127.0.0.1:7000")
	}
}

func TestGetListenAddr_custom(t *testing.T) {
	t.Setenv("LISTEN_ADDR", "0.0.0.0:8080")
	t.Setenv("DATA_DIR", "")
	t.Setenv("HOST_DATA_DIR", "")
	defer os.Unsetenv("LISTEN_ADDR")

	cfg, _ := Init("")
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
	var cfg AppConfig
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
```

- [ ] **Step 3: Verify tests pass**

Run: `cd /home/kev/work/projects/singbox_ui/server && go test ./internal/pkg/config/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add server/internal/pkg/config/config_test.go
git commit -m "refactor(config): update tests for inverted config dependencies and new DataDir default"
```

---

### Task 8: Full build and lint verification

- [ ] **Step 1: Build all**

Run: `cd /home/kev/work/projects/singbox_ui/server && go build ./...`
Expected: SUCCESS

- [ ] **Step 2: Run all tests**

Run: `cd /home/kev/work/projects/singbox_ui/server && go test ./...`
Expected: ALL PASS

- [ ] **Step 3: Run lint**

Run: `cd /home/kev/work/projects/singbox_ui/server && golangci-lint run ./...`
Expected: No errors (or only pre-existing ones unrelated to our changes)
