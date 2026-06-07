# Unified Application Configuration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate all application parameters (server, prober, speedtest, scheduler, subscription) into a single YAML config file with per-module Go structs.

**Architecture:** Each domain package defines its own `Config` struct. `internal/pkg/config` provides `AppConfig` (composing all sub-configs) + YAML loading + bootstrap (`DATA_DIR` env → `config.yaml`). Services receive their sub-config directly. Backwards-compatible: no config file → all defaults.

**Tech Stack:** Go 1.24+, `gopkg.in/yaml.v3` (already a dependency via Clash parser)

---

### Task 1: Define per-module config structs

**Files:**
- Modify: `server/internal/prober/models.go`
- Create: `server/internal/speedtest/models.go`
- Create: `server/internal/scheduler/models.go`
- Modify: `server/internal/subscription/models.go`

- [ ] **Step 1: Rename `ProberConfig` → `Config` in `server/internal/prober/models.go`**

Rename the struct, add `MaxRetries` field, update all references within the file.

```go
package prober

// Config holds configuration parameters for the prober engine.
type Config struct {
	Interval       int    `json:"interval" yaml:"interval" example:"30"`
	Timeout        int    `json:"timeout" yaml:"timeout" example:"5000"`
	Concurrent     int    `json:"concurrent" yaml:"concurrent" example:"5"`
	MaxResults     int    `json:"max_results" yaml:"max_results" example:"100"`
	MaxRetries     int    `json:"max_retries" yaml:"max_retries" example:"2"`
	BindAddress    string `json:"bind_address,omitempty" yaml:"bind_address" example:"192.168.1.100"`
	BindInterface  string `json:"bind_interface,omitempty" yaml:"bind_interface" example:"eth0"`
}
```

- [ ] **Step 2: Create `server/internal/speedtest/models.go`**

```go
package speedtest

// Config holds configuration parameters for the speed test service.
type Config struct {
	LatencyURL  string `json:"latency_url" yaml:"latency_url" example:"http://www.gstatic.com/generate_204"`
	DownloadURL string `json:"download_url" yaml:"download_url" example:"https://speed.cloudflare.com/__down?bytes=10000000"`
	Duration    int    `json:"duration" yaml:"duration" example:"10"` // seconds
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		LatencyURL:  "http://www.gstatic.com/generate_204",
		DownloadURL: "https://speed.cloudflare.com/__down?bytes=10000000",
		Duration:    10,
	}
}
```

- [ ] **Step 3: Create `server/internal/scheduler/models.go`**

```go
package scheduler

// Config holds configuration parameters for the scheduler.
type Config struct {
	Interval int `json:"interval" yaml:"interval" example:"60"` // seconds
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{Interval: 60}
}
```

- [ ] **Step 4: Add `Config` struct to `server/internal/subscription/models.go`**

Append before the closing of the file:

```go
// Config holds configuration parameters for subscription fetching.
type Config struct {
	InsecureTLS bool `json:"insecure_tls" yaml:"insecure_tls" example:"false"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{InsecureTLS: false}
}
```

- [ ] **Step 5: Run build to verify**

```bash
cd server && go build ./...
```
Expected: compilation errors because engine.go still uses `ProberConfig` (next task fixes it).

---

### Task 2: Update prober engine to use renamed `Config`

**Files:**
- Modify: `server/internal/prober/engine.go`

- [ ] **Step 1: Replace all `ProberConfig` → `Config` in engine.go and update field references**

Replace `config ProberConfig` → `config Config` in the `Prober` struct (line 53).
Replace `NewProber(config ProberConfig)` → `NewProber(config Config)` (line 67).
Update field refs: `ProbeInterval` → `Interval`, `ProbeTimeout` → `Timeout`, `ProbeConcurrent` → `Concurrent` throughout.

Key changes:

```go
// Line 53
type Prober struct {
	config    Config
	// ... rest unchanged
}

// Line 67
func NewProber(config Config) *Prober {

// Line 78-86
func DefaultProberConfig() Config {
	return Config{
		Interval:    30,
		Timeout:     5000,
		Concurrent:  5,
		MaxResults:  100,
		MaxRetries:  2,
	}
}

// Line 244
ticker := time.NewTicker(time.Duration(p.config.Interval) * time.Second)

// Line 327
Timeout: time.Duration(p.config.Timeout) * time.Millisecond,

// Lines 428-431
"probeInterval":   p.config.Interval,
"probeTimeout":    p.config.Timeout,
"probeConcurrent": p.config.Concurrent,
```

- [ ] **Step 2: Build and test**

```bash
cd server && go build ./...
```

---

### Task 3: Refactor `internal/pkg/config` — ServerConfig + AppConfig + YAML loading

**Files:**
- Modify: `server/internal/pkg/config/config.go`
- Modify: `server/internal/pkg/config/config_test.go`

- [ ] **Step 1: Rewrite `server/internal/pkg/config/config.go`**

Replace the file entirely with the new structure:

```go
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"singbox-config-service/internal/prober"
	"singbox-config-service/internal/scheduler"
	"singbox-config-service/internal/speedtest"
	"singbox-config-service/internal/subscription"

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
type AppConfig struct {
	Server       ServerConfig          `yaml:"server"`
	Prober       prober.Config         `yaml:"prober"`
	Speedtest    speedtest.Config      `yaml:"speedtest"`
	Scheduler    scheduler.Config      `yaml:"scheduler"`
	Subscription subscription.Config   `yaml:"subscription"`
}

// defaultAppConfig returns an AppConfig with all defaults set.
func defaultAppConfig() AppConfig {
	return AppConfig{
		Server: ServerConfig{
			ListenAddr: "127.0.0.1:7000",
		},
		Prober:       prober.DefaultConfig(),
		Speedtest:    speedtest.DefaultConfig(),
		Scheduler:    scheduler.DefaultConfig(),
		Subscription: subscription.DefaultConfig(),
	}
}

// Load reads a YAML config file and merges defaults for zero-valued fields.
func Load(path string) (*AppConfig, error) {
	cfg := defaultAppConfig()

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

	// Re-apply defaults for zero-valued fields after unmarshal.
	// This ensures partial configs still get sensible defaults.
	cfg = mergeDefaults(cfg)

	return &cfg, nil
}

// mergeDefaults fills zero-valued fields with defaults.
func mergeDefaults(cfg AppConfig) AppConfig {
	def := defaultAppConfig()

	if cfg.Server.ListenAddr == "" {
		cfg.Server.ListenAddr = def.Server.ListenAddr
	}
	if cfg.Prober.Interval == 0 {
		cfg.Prober.Interval = def.Prober.Interval
	}
	if cfg.Prober.Timeout == 0 {
		cfg.Prober.Timeout = def.Prober.Timeout
	}
	if cfg.Prober.Concurrent == 0 {
		cfg.Prober.Concurrent = def.Prober.Concurrent
	}
	if cfg.Prober.MaxResults == 0 {
		cfg.Prober.MaxResults = def.Prober.MaxResults
	}
	if cfg.Prober.MaxRetries == 0 {
		cfg.Prober.MaxRetries = def.Prober.MaxRetries
	}
	if cfg.Speedtest.LatencyURL == "" {
		cfg.Speedtest.LatencyURL = def.Speedtest.LatencyURL
	}
	if cfg.Speedtest.DownloadURL == "" {
		cfg.Speedtest.DownloadURL = def.Speedtest.DownloadURL
	}
	if cfg.Speedtest.Duration == 0 {
		cfg.Speedtest.Duration = def.Speedtest.Duration
	}
	if cfg.Scheduler.Interval == 0 {
		cfg.Scheduler.Interval = def.Scheduler.Interval
	}
	// InsecureTLS defaults to false (zero value), no override needed.

	return cfg
}

// Init bootstraps configuration. It resolves the data directory and config file path.
// If configPath is non-empty, it is used directly. Otherwise DATA_DIR/config.yaml is tried.
// If no config file exists, all defaults apply (backwards compatible).
func Init(configPath string) (*AppConfig, error) {
	// Resolve data directory (env or CWD)
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		workDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine working directory: %w", err)
		}
		if _, err := os.Stat(filepath.Join(workDir, "go.mod")); err == nil {
			dataDir = workDir
		} else if _, err := os.Stat(filepath.Join(workDir, "server", "go.mod")); err == nil {
			dataDir = filepath.Join(workDir, "server")
		} else {
			dataDir = workDir
		}
	}

	// Determine config file path
	path := configPath
	if path == "" {
		path = filepath.Join(dataDir, "config.yaml")
	}

	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	// If file was absent, path doesn't matter — we have defaults.
	// If user specified --config but it had no data_dir, apply the resolved dataDir.
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

	// Ensure singbox subdirectory exists.
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

- [ ] **Step 2: Rewrite `server/internal/pkg/config/config_test.go`**

Update tests to match the new `AppConfig` API:

```go
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
```

- [ ] **Step 3: Build and test**

```bash
cd server && go build ./... && go test ./internal/pkg/config/... -v
```
Expected: build succeeds, config tests pass.

---

### Task 4: Update prober service to accept `prober.Config` in constructor

**Files:**
- Modify: `server/internal/prober/service.go`

- [ ] **Step 1: Update `NewService` to accept `Config` param**

```go
// NewService creates a new Service with the given config, base directory and result saver.
func NewService(cfg Config, baseDir string, resultSaver ProbeResultSaver) *Service {
	return &Service{
		config:      cfg,
		baseDir:     baseDir,
		resultSaver: resultSaver,
	}
}
```

Remove the `DefaultProberConfig()` call from `NewService` — the config is now passed in from outside.

- [ ] **Step 2: Build**

```bash
cd server && go build ./...
```
Expected: compiles (main.go not yet updated, so prober.NewService call in main.go will error — that's fine, we fix main.go in a later task).

---

### Task 5: Update speedtest service to accept `speedtest.Config` and remove package constants

**Files:**
- Modify: `server/internal/speedtest/service.go`

- [ ] **Step 1: Update Service struct and constructor**

```go
// Service orchestrates speed tests against proxy nodes using sing-box instances.
type Service struct {
	cfg          Config
	tempRuntime  TempRuntime
	dockerCfg    *config.ServerConfig
	nodeProvider NodeProvider
	resultSaver  SpeedTestResultSaver
	state        *SpeedTestState
	mu           sync.Mutex
	cancel       context.CancelFunc
}

// NewService creates a new Service with the given TempRuntime and config.
func NewService(tempRuntime TempRuntime, speedtestCfg Config, serverCfg *config.ServerConfig) *Service {
	return &Service{
		cfg:         speedtestCfg,
		tempRuntime: tempRuntime,
		dockerCfg:   serverCfg,
		state:       &SpeedTestState{},
	}
}
```

The import for `"singbox-config-service/internal/pkg/config"` may already be present; keep it for the `*config.ServerConfig` reference.

- [ ] **Step 2: Replace all usage of package-level constants with `s.cfg` fields**

Replace these in the same file:

```go
// Line 211 (was s.cfg.GetSingboxDir())
dir := filepath.Join(s.dockerCfg.DataDir, "singbox", "speedtest")

// Line 246
req, _ := http.NewRequestWithContext(ctx, "GET", s.cfg.LatencyURL, nil)

// Line 262-263
dlClient := newProxyClient(proxyURL, time.Duration(s.cfg.Duration+5)*time.Second)
dlCtx, dlCancel := context.WithTimeout(ctx, time.Duration(s.cfg.Duration)*time.Second)

// Line 265
req, _ := http.NewRequestWithContext(dlCtx, "GET", s.cfg.DownloadURL, nil)
```

- [ ] **Step 3: Remove the package-level constants block**

Delete lines 24-31 (the `const` block with `speedTestLatencyURL`, `speedTestDownloadURL`, `speedTestDuration`).

- [ ] **Step 4: Update `runtimer_docker.go` and `runtimer_native.go` — rename `*config.Config` → `*config.AppConfig`**

**`server/internal/speedtest/runtimer_docker.go`:**
- Change `cfg *config.Config` → `cfg *config.AppConfig` in the struct field (line 21) and function signature (line 25).
- The `d.cfg.ResolveHostConfigDir(configPath)` call still works because `ResolveHostConfigDir` is a method on `*AppConfig`.

```go
type DockerTempRuntime struct {
	client *docker.Client
	cfg    *config.AppConfig
}

func NewTempRuntime(cfg *config.AppConfig) TempRuntime {
```

**`server/internal/speedtest/runtimer_native.go`:**
- Change `cfg *config.Config` → `cfg *config.AppConfig` in function signature (line 25).
- `cfg.GetSingboxBinPath()` is still a method on `*AppConfig`, no change needed.

```go
func NewTempRuntime(cfg *config.AppConfig) TempRuntime {
```

- [ ] **Step 5: Build**

```bash
cd server && go build ./...
```
Expected: compiles (main.go still uses old constructors — will be fixed in a later task).

---

### Task 6: Update scheduler constructor to accept `scheduler.Config`

**Files:**
- Modify: `server/internal/scheduler/service.go`

- [ ] **Step 1: Update constructor**

```go
// New creates a new Scheduler with the given config, updater and container manager.
func New(cfg Config, subUpdater SubscriptionUpdater, containerMgr ContainerManager) *Scheduler {
	return &Scheduler{
		subUpdater:   subUpdater,
		containerMgr: containerMgr,
		interval:     time.Duration(cfg.Interval) * time.Second,
	}
}
```

Remove the hardcoded `interval: 60 * time.Second` from the constructor body.

- [ ] **Step 2: Build**

```bash
cd server && go build ./...
```

---

### Task 7: Update subscription service to accept `subscription.Config`

**Files:**
- Modify: `server/internal/subscription/service.go`

- [ ] **Step 1: Add `cfg` field to `Service` struct and update constructor**

```go
// Service manages subscription operations including fetching, adding,
// updating, deleting, and refreshing proxy node subscriptions.
type Service struct {
	store *FileStore
	cfg   Config
}

// NewService creates a new Service backed by the given FileStore with config.
func NewService(store *FileStore, cfg Config) *Service {
	return &Service{store: store, cfg: cfg}
}
```

- [ ] **Step 2: Replace `allowInsecureTLS` to use config instead of `os.Getenv`**

```go
// allowInsecureTLS returns true if the config allows insecure TLS.
func (s *Service) allowInsecureTLS() bool {
	return s.cfg.InsecureTLS
}
```

Remove the `"os"` import if it's no longer used elsewhere in the file. Check with `grep 'os\.'` in the file.

- [ ] **Step 3: Build**

```bash
cd server && go build ./...
```

---

### Task 8: Update sing-box runtime files to use `*config.AppConfig`

**Files:**
- Modify: `server/internal/singbox/runtime_docker.go`
- Modify: `server/internal/singbox/runtime_native.go`
- Modify: `server/internal/singbox/service.go`

- [ ] **Step 1: Rename `*config.Config` → `*config.AppConfig` in singbox runtime files**

**`server/internal/singbox/runtime_docker.go`:**
```go
// Line 21: cfg *config.Config → cfg *config.AppConfig
// Line 25: func NewRuntime(cfg *config.AppConfig) (Runtime, error) {
```

**`server/internal/singbox/runtime_native.go`:**
```go
// Line 32: func NewRuntime(cfg *config.AppConfig) (Runtime, error) {
```

- [ ] **Step 2: Rename in `server/internal/singbox/service.go`**

```go
// Line 17: cfg *config.AppConfig
// Line 21: func NewService(runtime Runtime, cfg *config.AppConfig) *Service {
```

- [ ] **Step 3: Build**

```bash
cd server && go build ./...
```

---

### Task 9: Update `main.go` — wire new constructors, add `--config` flag

**Files:**
- Modify: `server/main.go`

- [ ] **Step 1: Replace CLI flags and update initialization in `main()`**

```go
func main() {
	// Parse CLI flags
	configPath := flag.String("config", "", "Path to config file (default: DATA_DIR/config.yaml)")
	flag.Parse()

	// Initialize config
	cfg, err := config.Init(*configPath)
	if err != nil {
		log.Printf("Warning: Failed to initialize config: %v", err)
		log.Println("Some features may not work properly")
	}

	// Create domain services
	rt, err := singbox.NewRuntime(cfg)
	if err != nil {
		log.Printf("Warning: Failed to create sing-box runtime: %v", err)
		log.Println("sing-box features will not be available")
		rt = &singbox.NoopRuntime{}
	}
	singboxSvc := singbox.NewService(rt, cfg)
	wgSvc := wireguard.NewService(cfg.GetDataDir())
	warpSvc := warp.NewService(cfg.GetDataDir())
	certSvc := certificate.NewService(cfg.GetSingboxDir())
	subStore := subscription.NewFileStore(cfg.GetDataDir())
	subSvc := subscription.NewService(subStore, cfg.Subscription)
	proberSvc := prober.NewService(cfg.Prober, cfg.GetDataDir(), subSvc)
	speedtestSvc := speedtest.NewService(speedtest.NewTempRuntime(cfg), cfg.Speedtest, &cfg.Server)

	// Create auto-update scheduler
	sched := scheduler.New(cfg.Scheduler, subSvc, nil)

	// ... rest of main() unchanged (handlers, routes, etc.) ...
```

- [ ] **Step 2: Update `NewRuntime` calls (singbox, speedtest)**

The `singbox.NewRuntime(cfg)` and `speedtest.NewTempRuntime(cfg)` take the full `*AppConfig` now. They must compile. Check that `singbox/runtime_docker.go` and `speedtest/runtimer_*.go` use methods that still exist on `*AppConfig` (they use `GetDataDir()`, `GetSingboxBinPath()`, etc. — these methods exist on `*AppConfig`).

- [ ] **Step 3: Build**

```bash
cd server && go build ./...
```
Expected: clean build.

---

### Task 10: Update tests in affected packages

**Files:**
- Modify: `server/internal/prober/engine_test.go` (field name updates)
- Modify: `server/internal/prober/service_test.go` (constructor change)
- Modify: `server/internal/speedtest/service_test.go` (constructor change)
- Modify: `server/internal/scheduler/service_test.go` (constructor change)
- Modify: `server/internal/subscription/service_test.go` (constructor change)

- [ ] **Step 1: Fix `engine_test.go` — update `DefaultProberConfig()` calls**

Replace `DefaultProberConfig()` → `DefaultConfig()` and update field checks (`ProbeInterval` → `Interval`, `ProbeTimeout` → `Timeout`, `ProbeConcurrent` → `Concurrent`). The test file has ~20 references to `DefaultProberConfig()`.

Replace all:
```go
p := NewProber(DefaultProberConfig())
```
with:
```go
p := NewProber(DefaultConfig())
```

And field assertions:
```go
// Was:
if p.config.ProbeInterval != 30 {
if p.config.ProbeTimeout != 5000 {

// Now:
if p.config.Interval != 30 {
if p.config.Timeout != 5000 {
```

- [ ] **Step 2: Fix `service_test.go` for prober — update `NewService` calls**

Find all `NewService(baseDir, resultSaver)` calls and add `prober.DefaultConfig()` as first argument:

```go
// Was:
svc := NewService(dir, mockSaver)
// Now:
svc := NewService(DefaultConfig(), dir, mockSaver)
```

- [ ] **Step 3: Fix `speedtest/service_test.go` — update `NewService` calls**

Find all `NewService(tempRuntime, cfg)` calls and change to:

```go
// Was:
svc := NewService(mockRT, cfg)
// Now:
svc := NewService(mockRT, DefaultConfig(), &cfg.Server)
```

Where `cfg` was `*config.Config` it is now `*config.AppConfig`.

- [ ] **Step 4: Fix `scheduler/service_test.go` — update `New` calls**

```go
// Was:
sched := New(subUpdater, nil)
// Now:
sched := New(DefaultConfig(), subUpdater, nil)
```

- [ ] **Step 5: Fix `subscription/service_test.go` — update `NewService` calls**

```go
// Was:
svc := NewService(store)
// Now:
svc := NewService(store, DefaultConfig())
```

- [ ] **Step 6: Run all tests**

```bash
cd server && go test ./... -count=1 2>&1 | tail -50
```
Expected: all tests pass.

---

### Task 11: Update docker-compose.yml and create config.example.yaml

**Files:**
- Modify: `docker-compose.yml`
- Create: `config.example.yaml`

- [ ] **Step 1: Update `docker-compose.yml`**

Remove `LISTEN_ADDR` from environment (now lives in config.yaml). Keep `DATA_DIR` and `TZ`. Add a volume mount for config.yaml:

```yaml
services:
  singbox-ui:
    build: .
    container_name: singbox-ui
    restart: unless-stopped
    network_mode: host
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/home/data
    environment:
      - DATA_DIR=/home/data
      - TZ=Asia/Shanghai
    # Optional: mount a custom config file
    # - ./config.yaml:/home/data/config.yaml
```

- [ ] **Step 2: Create `config.example.yaml`**

```yaml
# singbox-ui configuration file
# Place this file at DATA_DIR/config.yaml or pass via --config flag.
# All fields are optional; defaults are shown below.

server:
  # listen_addr: "127.0.0.1:7000"    # HTTP server listen address
  # data_dir: ""                      # Data directory (overrides DATA_DIR env)
  # host_data_dir: ""                 # Host path for DATA_DIR (Docker path mapping)
  # singbox_bin_path: ""              # Path to sing-box binary (native mode)
  # serve_dashboard: false            # Serve embedded frontend

prober:
  # interval: 30                      # Probe interval (seconds)
  # timeout: 5000                     # TCP dial timeout (ms)
  # concurrent: 5                     # Max concurrent probes
  # max_results: 100                  # Ring buffer size for history
  # max_retries: 2                    # Retries on probe failure
  # bind_address: ""                  # Local IP to bind probes to (bypass tunnel)
  # bind_interface: ""                # Network interface to bind probes to (requires root)

speedtest:
  # latency_url: "http://www.gstatic.com/generate_204"
  # download_url: "https://speed.cloudflare.com/__down?bytes=10000000"
  # duration: 10                      # Download test duration (seconds)

scheduler:
  # interval: 60                      # Subscription auto-update check interval (seconds)

subscription:
  # insecure_tls: false               # Allow insecure TLS when fetching subscriptions
```

- [ ] **Step 3: Build and final check**

```bash
cd server && go build ./...
```

---

### Task 12: Final verification

**Files:**
- All modified files

- [ ] **Step 1: Full build**

```bash
cd server && go build ./...
```
Expected: exit code 0, no output.

- [ ] **Step 2: Run linter**

```bash
export PATH="$(go env GOPATH)/bin:$PATH" && golangci-lint run ./...
```
Expected: only pre-existing warnings (speedtest/runtimer_native.go, singbox/runtime_native.go).

- [ ] **Step 3: Full test suite**

```bash
cd server && go test ./... -count=1
```
Expected: all tests pass.

- [ ] **Step 4: Commit all changes**

```bash
git add -A && git commit -m "feat: unified application configuration via YAML

- Rename ProberConfig → Config, add MaxRetries, bind options
- Create speedtest.Config, scheduler.Config, subscription.Config
- Refactor internal/pkg/config: ServerConfig + AppConfig + YAML loading
- Update all service constructors to receive their sub-config
- Replace --dashboard/--singbox-bin flags with --config
- Add config.example.yaml
- Backwards-compatible: no config file = all defaults"
```
