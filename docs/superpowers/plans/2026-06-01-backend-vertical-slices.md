# Backend Vertical Slices Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate backend from layered architecture (handlers/ → services/) to vertical slices with 100% public API test coverage.

**Architecture:** 8 domain packages in `internal/`, shared infrastructure in `internal/pkg/`, all cross-domain communication through interfaces. No global singletons.

**Tech Stack:** Go 1.24, Gin, Docker SDK, `github.com/stretchr/testify` (for test assertions)

**Key Constraint:** Every exported function, method, type, and interface in every new package MUST have tests. The final state after step 8 must have `go test -race -cover ./...` showing ≥95% coverage for all new packages.

---

## File Map

### New files to create

```
server/internal/
├── pkg/
│   ├── types/
│   │   ├── proxy_node.go         # ProxyNode, ProbeNode, ProbeResult, etc.
│   │   ├── sanizite_tag.go       # SanitizeTag(), ResolveUserAgent()
│   │   ├── constants.go          # PredefinedUserAgents, proxyOutboundTypes, blockedSubscriptionPrefixes
│   │   └── sanizite_tag_test.go  # Tests for SanitizeTag, ResolveUserAgent
│   │   └── proxy_node_test.go    # Tests for ProxyNode helpers
│   ├── config/
│   │   ├── config.go             # Config struct, Init(), path accessors
│   │   └── config_test.go        # Tests for config resolution
│   │   └── config_unix_test.go   # Unix-specific path tests
│   └── docker/
│       ├── client.go             # Docker client struct, NewClient()
│       ├── interfaces.go         # ContainerAPI interface
│       ├── models.go             # ContainerInfo struct
│       ├── client_test.go        # Tests for client methods (with mock server)
│       └── mock_test.go          # MockContainerAPI for other packages
├── singbox/
│   ├── handler.go                # HTTP handlers
│   ├── register.go               # RegisterRoutes()
│   ├── service.go                # Business logic
│   ├── models.go                 # NamedConfigInfo
│   ├── interfaces.go             # ContainerManager interface
│   ├── handler_test.go           # Handler tests
│   ├── service_test.go           # Service tests (with docker mock)
│   └── models_test.go            # Model tests
├── subscription/
│   ├── handler.go                # HTTP handlers
│   ├── register.go               # RegisterRoutes()
│   ├── service.go                # CRUD + fetch
│   ├── store.go                  # File-based persistence
│   ├── interfaces.go             # SubscriptionUpdater, NodeProvider, ProbeResultSaver, SpeedTestResultSaver
│   ├── parser_vmess.go           # VMess URL parser
│   ├── parser_vless.go           # VLESS URL parser
│   ├── parser_trojan.go          # Trojan URL parser
│   ├── parser_ss.go              # Shadowsocks URL parser
│   ├── parser_clash.go           # Clash YAML parser
│   ├── models.go                 # SubscriptionEntry, SubscriptionData
│   ├── handler_test.go           # Handler tests
│   ├── service_test.go           # Service tests (mock store + HTTP)
│   ├── parser_vmess_test.go      # VMess parser tests
│   ├── parser_vless_test.go      # VLESS parser tests
│   ├── parser_trojan_test.go     # Trojan parser tests
│   ├── parser_ss_test.go         # Shadowsocks parser tests
│   ├── parser_clash_test.go      # Clash YAML parser tests
│   └── store_test.go             # In-memory store tests
├── prober/
│   ├── handler.go                # HTTP handlers
│   ├── register.go               # RegisterRoutes()
│   ├── engine.go                 # Prober struct: Start, Stop, probeLoop, probeNode
│   ├── service.go                # CRUD + lifecycle
│   ├── models.go                 # ProberConfig
│   ├── interfaces.go             # ProbeResultSaver (if used)
│   ├── handler_test.go           # Handler tests
│   ├── engine_test.go            # Engine lifecycle tests
│   └── service_test.go           # CRUD tests
├── speedtest/
│   ├── handler.go                # HTTP handlers
│   ├── register.go               # RegisterRoutes()
│   ├── service.go                # testOneNode, runSpeedTest
│   ├── models.go                 # SpeedTestResult, SpeedTestState
│   ├── handler_test.go           # Handler tests
│   └── service_test.go           # Service tests (docker mock)
├── certificate/
│   ├── handler.go                # HTTP handlers
│   ├── register.go               # RegisterRoutes()
│   ├── service.go                # GenerateSelfSignedCert, GetCertificateInfo
│   ├── models.go                 # CertificateInfo
│   ├── handler_test.go           # Handler tests
│   └── service_test.go           # Certificate generation tests
├── wireguard/
│   ├── handler.go                # HTTP handlers
│   ├── register.go               # RegisterRoutes()
│   ├── service.go                # GenerateWireGuardKeysWithCache, GetKeysCache, SaveClientConfig, GetPublicIP
│   ├── models.go                 # WireGuardKeyPair, KeyCacheEntry, ClientConfigFile
│   ├── handler_test.go           # Handler tests
│   └── service_test.go           # Key generation + cache tests
└── warp/
    ├── handler.go                # HTTP handlers
    ├── register.go               # RegisterRoutes()
    ├── service.go                # RegisterWarpDevice, BindWarpLicense, BuildWarpOutbound
    ├── scanner.go                # ScanWarpEndpoints, warpHandshakeProbe
    ├── models.go                 # WarpRecord, WarpEndpointResult, WarpScanConfig
    ├── handler_test.go           # Handler tests
    ├── service_test.go           # Service tests (HTTP mock)
    └── scanner_test.go           # Scanner tests
```

### Modified files

| File | Change |
|------|--------|
| `server/main.go` | Rewrite: new DI, new route registration, remove global init |
| `server/init.go` | Delete (logic distributed to config.Init + constructors) |
| `server/handlers/*.go` | Delete (all 7 files) |
| `server/services/*.go` | Delete (all 13 files — after migration complete) |

---

## Step 1: Shared Package — `internal/pkg/`

### Task 1.1: Create `internal/pkg/types/`

**Files:**
- Create: `server/internal/pkg/types/proxy_node.go`
- Create: `server/internal/pkg/types/sanitize_tag.go`
- Create: `server/internal/pkg/types/constants.go`
- Test: `server/internal/pkg/types/sanitize_tag_test.go`
- Test: `server/internal/pkg/types/proxy_node_test.go`

- [ ] **Step 1: Write the failing test for SanitizeTag**

```go
// server/internal/pkg/types/sanitize_tag_test.go
package types

import (
	"testing"
)

func TestSanitizeTag_ipv4(t *testing.T) {
	got := SanitizeTag("vmess", "1.2.3.4", 443)
	want := "vmess-1_2_3_4-443"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestSanitizeTag_ipv6(t *testing.T) {
	got := SanitizeTag("vless", "2001:db8::1", 8080)
	want := "vless-2001_db8__1-8080"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestSanitizeTag_specialChars(t *testing.T) {
	got := SanitizeTag("ss", "host-name.com", 8388)
	want := "ss-host_name_com-8388"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestSanitizeTag_emptyAddress(t *testing.T) {
	got := SanitizeTag("direct", "", 0)
	want := "direct--0"
	if got != want {
		t.Errorf("SanitizeTag() = %q, want %q", got, want)
	}
}

func TestResolveUserAgent_predefined(t *testing.T) {
	got := ResolveUserAgent("clash-verge")
	want := "clash-verge/v2.4.0"
	if got != want {
		t.Errorf("ResolveUserAgent() = %q, want %q", got, want)
	}
}

func TestResolveUserAgent_custom(t *testing.T) {
	got := ResolveUserAgent("MyCustomUA/1.0")
	want := "MyCustomUA/1.0"
	if got != want {
		t.Errorf("ResolveUserAgent() = %q, want %q", got, want)
	}
}

func TestResolveUserAgent_empty(t *testing.T) {
	got := ResolveUserAgent("")
	want := PredefinedUserAgents["default"]
	if got != want {
		t.Errorf("ResolveUserAgent() = %q, want %q", got, want)
	}
}
```

Run: `go test ./internal/pkg/types/ -v`
Expected: FAIL (package does not exist yet)

- [ ] **Step 2: Create package files**

```go
// server/internal/pkg/types/sanitize_tag.go
package types

import (
	"fmt"
	"strings"
)

var PredefinedUserAgents = map[string]string{
	"clash-verge": "clash-verge/v2.4.0",
	"clash-meta":  "ClashMeta/v1.18.0",
	"v2rayn":      "v2rayN/6.0",
	"v2rayng":     "v2rayNG/1.8.0",
	"default":     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

func SanitizeTag(protocol, address string, port int) string {
	safeAddress := strings.ReplaceAll(address, ".", "_")
	safeAddress = strings.ReplaceAll(safeAddress, ":", "_")
	safeAddress = strings.ReplaceAll(safeAddress, "-", "_")
	return fmt.Sprintf("%s-%s-%d", protocol, safeAddress, port)
}

func ResolveUserAgent(ua string) string {
	if ua == "" {
		return PredefinedUserAgents["default"]
	}
	if predefined, ok := PredefinedUserAgents[ua]; ok {
		return predefined
	}
	return ua
}
```

```go
// server/internal/pkg/types/proxy_node.go
package types

type ProxyNode struct {
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Address  string                 `json:"address"`
	Port     int                    `json:"port"`
	Settings map[string]interface{} `json:"settings"`
	Outbound map[string]interface{} `json:"outbound"`
	Latency     int64   `json:"latency,omitempty"`
	Online      bool    `json:"online,omitempty"`
	LastProbe   string  `json:"last_probe,omitempty"`
	SuccessRate int     `json:"success_rate,omitempty"`
	SpeedKBps   float64 `json:"speed_kbps,omitempty"`
}

type ProbeNode struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
}

type ProbeResult struct {
	NodeTag     string  `json:"nodeTag"`
	Protocol    string  `json:"protocol"`
	Address     string  `json:"address"`
	Port        int     `json:"port"`
	Latency     int64   `json:"latency"`
	Status      string  `json:"status"`
	LastProbe   string  `json:"lastProbe"`
	FailCount   int     `json:"failCount"`
	SuccessRate float64 `json:"successRate"`
}

type ProbeResultUpdate struct {
	Tag         string `json:"tag"`
	Latency     int64  `json:"latency"`
	Online      bool   `json:"online"`
	LastProbe   string `json:"last_probe"`
	SuccessRate int    `json:"success_rate"`
}

type SpeedTestResult struct {
	Tag       string  `json:"tag"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	LatencyMs int64   `json:"latency_ms"`
	SpeedKBps float64 `json:"speed_kbps"`
	Error     string  `json:"error,omitempty"`
	TestedAt  string  `json:"tested_at,omitempty"`
}

type SpeedTestUpdate struct {
	Tag       string  `json:"tag"`
	Latency   int64   `json:"latency"`
	SpeedKBps float64 `json:"speed_kbps"`
	Online    bool    `json:"online"`
	LastProbe string  `json:"last_probe"`
}
```

```go
// server/internal/pkg/types/constants.go
package types

import "net/netip"

var proxyOutboundTypes = map[string]bool{
	"vless": true, "vmess": true, "trojan": true, "shadowsocks": true,
	"hysteria2": true, "tuic": true, "wireguard": true, "socks": true,
	"http": true, "ssh": true, "anytls": true, "shadowtls": true, "naive": true,
}

var BlockedSubscriptionPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("ff00::/8"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func IsProxyOutboundType(t string) bool {
	return proxyOutboundTypes[t]
}
```

```go
// server/internal/pkg/types/proxy_node_test.go
package types

import "testing"

func TestIsProxyOutboundType_known(t *testing.T) {
	if !IsProxyOutboundType("vless") {
		t.Error("IsProxyOutboundType('vless') = false, want true")
	}
}

func TestIsProxyOutboundType_unknown(t *testing.T) {
	if IsProxyOutboundType("freedom") {
		t.Error("IsProxyOutboundType('freedom') = true, want false")
	}
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test ./internal/pkg/types/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add server/internal/pkg/types/
git commit -m "feat(internal): add shared types package with SanitizeTag, ProxyNode, constants"
```

### Task 1.2: Create `internal/pkg/config/`

**Files:**
- Create: `server/internal/pkg/config/config.go`
- Test: `server/internal/pkg/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/pkg/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit_defaultPath(t *testing.T) {
	os.Clearenv()
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	// Create go.mod to simulate server directory
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

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
```

- [ ] **Step 2: Create config package**

```go
// server/internal/pkg/config/config.go
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	dataDir       string
	hostDataDir   string
	listenAddr    string
	singboxDir    string
}

func Init() (*Config, error) {
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

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "127.0.0.1:7000"
	}
	hostDataDir := os.Getenv("HOST_DATA_DIR")

	cfg := &Config{
		dataDir:     dataDir,
		hostDataDir: hostDataDir,
		listenAddr:  listenAddr,
		singboxDir:  filepath.Join(dataDir, "singbox"),
	}

	if err := os.MkdirAll(cfg.singboxDir, 0755); err != nil {
		log.Printf("Warning: failed to create singbox directory: %v", err)
	}

	return cfg, nil
}

func (c *Config) GetDataDir() string {
	return c.dataDir
}

func (c *Config) GetSingboxDir() string {
	return c.singboxDir
}

func (c *Config) GetListenAddr() string {
	return c.listenAddr
}

func (c *Config) ResolveHostConfigDir(containerPath string) (string, error) {
	if c.hostDataDir == "" {
		return "", fmt.Errorf("HOST_DATA_DIR environment variable is not set")
	}

	rel, err := filepath.Rel(c.dataDir, containerPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is not under DATA_DIR %s", containerPath, c.dataDir)
	}
	return filepath.Join(c.hostDataDir, rel), nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/pkg/config/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add server/internal/pkg/config/
git commit -m "feat(internal): add config package with path resolution and env vars"
```

### Task 1.3: Create `internal/pkg/docker/`

**Files:**
- Create: `server/internal/pkg/docker/interfaces.go`
- Create: `server/internal/pkg/docker/models.go`
- Create: `server/internal/pkg/docker/client.go`
- Test: `server/internal/pkg/docker/client_test.go`

- [ ] **Step 1: Write the test (mock Docker API server)**

```go
// server/internal/pkg/docker/client_test.go
package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	client.Close()
}

func TestContainerInfo(t *testing.T) {
	info := ContainerInfo{
		Name:        "test",
		ContainerID: "abc123",
		State:       "running",
		Status:      "Up 1 hour",
		Created:     1000,
	}
	if info.Name != "test" {
		t.Errorf("ContainerInfo.Name = %q, want %q", info.Name, "test")
	}
}

func TestContainerAPI_interface(t *testing.T) {
	// Compile-time check: *Client implements ContainerAPI
	var _ ContainerAPI = (*Client)(nil)
}
```

Run: `go test ./internal/pkg/docker/ -v`
Expected: FAIL (package does not exist)

- [ ] **Step 2: Create docker package**

```go
// server/internal/pkg/docker/interfaces.go
package docker

import (
	"context"
	"io"
)

type ContainerAPI interface {
	ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (containerID string, err error)
	ContainerStart(ctx context.Context, containerID string) error
	ContainerStop(ctx context.Context, containerID string, timeout *int) error
	ContainerRemove(ctx context.Context, containerID string, force bool) error
	ContainerLogs(ctx context.Context, containerID string, tail string) (string, error)
	ContainerInspect(ctx context.Context, containerID string) (state string, err error)
	ImagePull(ctx context.Context, image string) (io.ReadCloser, error)
	ImageList(ctx context.Context, image string) (bool, error)
	Close() error
}
```

```go
// server/internal/pkg/docker/models.go
package docker

type ContainerInfo struct {
	Name        string `json:"name"`
	ContainerID string `json:"container_id"`
	State       string `json:"state"`
	Status      string `json:"status"`
	Created     int64  `json:"created"`
}
```

```go
// server/internal/pkg/docker/client.go
package docker

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Client struct {
	cli *client.Client
	ctx context.Context
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

func (d *Client) Close() error {
	if d.cli != nil {
		return d.cli.Close()
	}
	return nil
}

// EnsureImage ensures the image exists, trying load from tar first then pull
func (d *Client) EnsureImage(imageName, tarPath string) error {
	log.Printf("Checking if image %s exists...", imageName)

	images, err := d.cli.ImageList(d.ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageName)),
	})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}
	if len(images) > 0 {
		log.Printf("Image %s already exists", imageName)
		return nil
	}

	// Try loading from tar
	if tarPath != "" {
		if _, err := os.Stat(tarPath); err == nil {
			if err := d.loadImageFromFile(tarPath, imageName); err == nil {
				return nil
			} else {
				log.Printf("Embedded image load failed: %v, falling back to pull", err)
			}
		}
	}

	// Pull from registry
	log.Printf("Pulling image %s...", imageName)
	reader, err := d.cli.ImagePull(d.ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}
	log.Printf("Image %s pulled successfully", imageName)
	return nil
}

func (d *Client) loadImageFromFile(tarPath, imageName string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := d.cli.ImageLoad(d.ctx, file, true)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	// Re-tag CI temp tag if needed
	images, err := d.cli.ImageList(d.ctx, types.ImageListOptions{})
	if err == nil {
		for _, img := range images {
			for _, tag := range img.RepoTags {
				if strings.HasPrefix(tag, "singbox:") {
					if err := d.cli.ImageTag(d.ctx, img.ID, imageName); err != nil {
						log.Printf("Warning: failed to re-tag image: %v", err)
					}
					break
				}
			}
		}
	}

	// Verify
	verifyImages, err := d.cli.ImageList(d.ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageName)),
	})
	if err != nil || len(verifyImages) == 0 {
		return fmt.Errorf("image loaded but not found under expected tag %s", imageName)
	}

	os.Remove(tarPath)
	log.Printf("Image loaded from %s successfully", tarPath)
	return nil
}

func (d *Client) CreateContainer(config *container.Config, hostConfig *container.HostConfig, name string) (string, error) {
	resp, err := d.cli.ContainerCreate(d.ctx, config, hostConfig, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return resp.ID, nil
}

func (d *Client) StartContainer(containerID string) error {
	if err := d.cli.ContainerStart(d.ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

func (d *Client) StopContainer(containerID string, timeout *int) error {
	stopOptions := container.StopOptions{Timeout: timeout}
	if err := d.cli.ContainerStop(d.ctx, containerID, stopOptions); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}
	return nil
}

func (d *Client) RemoveContainer(containerID string, force bool) error {
	if err := d.cli.ContainerRemove(d.ctx, containerID, container.RemoveOptions{Force: force}); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}
	return nil
}

func (d *Client) ContainerLogs(containerID, tail string) (string, error) {
	if tail == "" {
		tail = "100"
	}
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	}
	reader, err := d.cli.ContainerLogs(d.ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, reader)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	logs := stdout.String()
	if stderr.Len() > 0 {
		logs += "\n--- STDERR ---\n" + stderr.String()
	}
	return logs, nil
}

func (d *Client) ContainerInspect(containerName string) (state string, err error) {
	containers, err := d.cli.ContainerList(d.ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	if len(containers) == 0 {
		return "", nil
	}
	return containers[0].State, nil
}

func (d *Client) ListContainers(prefix string) ([]ContainerInfo, error) {
	containers, err := d.cli.ContainerList(d.ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", prefix)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		result = append(result, ContainerInfo{
			Name:        name,
			ContainerID: c.ID[:12],
			State:       c.State,
			Status:      c.Status,
			Created:     c.Created,
		})
	}
	return result, nil
}

// ImageList checks if an image exists locally
func (d *Client) ImageList(ctx context.Context, image string) (bool, error) {
	images, err := d.cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", image)),
	})
	if err != nil {
		return false, err
	}
	return len(images) > 0, nil
}

// ImagePull pulls an image
func (d *Client) ImagePull(ctx context.Context, image string) (io.ReadCloser, error) {
	return d.cli.ImagePull(ctx, image, types.ImagePullOptions{})
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/pkg/docker/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add server/internal/pkg/docker/
git commit -m "feat(internal): add docker package with ContainerAPI interface and client"
```

---

## Step 2: Independent Domain Slices

### Task 2.1: Create `internal/certificate/`

**Files:**
- Create: `server/internal/certificate/service.go`
- Create: `server/internal/certificate/models.go`
- Create: `server/internal/certificate/handler.go`
- Create: `server/internal/certificate/register.go`
- Test: `server/internal/certificate/service_test.go`
- Test: `server/internal/certificate/handler_test.go`

- [ ] **Step 1: Write service tests**

```go
// server/internal/certificate/service_test.go
package certificate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	svc := NewService(t.TempDir())

	info, err := svc.GenerateSelfSignedCert("example.com", 30)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}

	if info.CommonName != "example.com" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "example.com")
	}
	if info.CertPath == "" || info.KeyPath == "" {
		t.Error("CertPath or KeyPath is empty")
	}
	if info.Fingerprint == "" {
		t.Error("Fingerprint is empty")
	}
	if info.ValidFrom == "" || info.ValidTo == "" {
		t.Error("ValidFrom or ValidTo is empty")
	}

	// Verify files exist
	if _, err := os.Stat(info.CertPath); os.IsNotExist(err) {
		t.Errorf("cert file not created: %s", info.CertPath)
	}
	if _, err := os.Stat(info.KeyPath); os.IsNotExist(err) {
		t.Errorf("key file not created: %s", info.KeyPath)
	}
}

func TestGenerateSelfSignedCert_defaultDomain(t *testing.T) {
	svc := NewService(t.TempDir())
	info, err := svc.GenerateSelfSignedCert("", 30)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}
	if info.CommonName != "localhost" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "localhost")
	}
}

func TestGenerateSelfSignedCert_defaultDays(t *testing.T) {
	svc := NewService(t.TempDir())
	info, err := svc.GenerateSelfSignedCert("test.com", 0)
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}
	if info.CommonName != "test.com" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "test.com")
	}
}

func TestGetCertificateInfo(t *testing.T) {
	svc := NewService(t.TempDir())
	genInfo, _ := svc.GenerateSelfSignedCert("test.com", 30)

	info, err := svc.GetCertificateInfo(genInfo.CertPath)
	if err != nil {
		t.Fatalf("GetCertificateInfo() error = %v", err)
	}
	if info.CommonName != "test.com" {
		t.Errorf("CommonName = %q, want %q", info.CommonName, "test.com")
	}
}

func TestGetCertificateInfo_notFound(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GetCertificateInfo("/nonexistent/cert.pem")
	if err == nil {
		t.Error("GetCertificateInfo() expected error, got nil")
	}
}

func TestCertificateExists(t *testing.T) {
	svc := NewService(t.TempDir())
	if svc.CertificateExists() {
		t.Error("CertificateExists() = true before generating cert")
	}
	svc.GenerateSelfSignedCert("test.com", 30)
	if !svc.CertificateExists() {
		t.Error("CertificateExists() = false after generating cert")
	}
}
```

- [ ] **Step 2: Create certificate package**

```go
// server/internal/certificate/service.go
package certificate

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Service struct {
	certDir string
}

func NewService(certDir string) *Service {
	return &Service{certDir: certDir}
}

func (s *Service) GenerateSelfSignedCert(domain string, validDays int) (*CertificateInfo, error) {
	if domain == "" {
		domain = "localhost"
	}
	if validDays <= 0 {
		validDays = 365
	}

	if err := os.MkdirAll(s.certDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create certificate directory: %w", err)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, validDays)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Sing-box UI Self-Signed"},
			CommonName:   domain,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(domain); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{domain}
	}
	if domain != "localhost" {
		template.DNSNames = append(template.DNSNames, "localhost")
	}
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	certPath := filepath.Join(s.certDir, "cert.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert.pem: %w", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return nil, fmt.Errorf("failed to write cert.pem: %w", err)
	}

	keyPath := filepath.Join(s.certDir, "key.pem")
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create key.pem: %w", err)
	}
	defer keyOut.Close()
	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, fmt.Errorf("failed to write key.pem: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	fingerprint := fmt.Sprintf("%X", cert.Raw[:20])

	return &CertificateInfo{
		CertPath:    certPath,
		KeyPath:     keyPath,
		CommonName:  domain,
		ValidFrom:   notBefore.Format(time.RFC3339),
		ValidTo:     notAfter.Format(time.RFC3339),
		Fingerprint: fingerprint[:40],
	}, nil
}

func (s *Service) GetCertificateInfo(certPath string) (*CertificateInfo, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	keyPath := filepath.Join(filepath.Dir(certPath), "key.pem")
	fingerprint := fmt.Sprintf("%X", cert.Raw[:20])
	return &CertificateInfo{
		CertPath:    certPath,
		KeyPath:     keyPath,
		CommonName:  cert.Subject.CommonName,
		ValidFrom:   cert.NotBefore.Format(time.RFC3339),
		ValidTo:     cert.NotAfter.Format(time.RFC3339),
		Fingerprint: fingerprint[:40],
	}, nil
}

func (s *Service) CertificateExists() bool {
	certPath := filepath.Join(s.certDir, "cert.pem")
	keyPath := filepath.Join(s.certDir, "key.pem")
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)
	return certErr == nil && keyErr == nil
}
```

```go
// server/internal/certificate/models.go
package certificate

type CertificateInfo struct {
	CertPath    string `json:"cert_path"`
	KeyPath     string `json:"key_path"`
	CommonName  string `json:"common_name"`
	ValidFrom   string `json:"valid_from"`
	ValidTo     string `json:"valid_to"`
	Fingerprint string `json:"fingerprint"`
}
```

```go
// server/internal/certificate/handler.go
package certificate

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GenerateSelfSignedCert(c *gin.Context) {
	var req struct {
		Domain    string `json:"domain"`
		ValidDays int    `json:"valid_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	info, err := h.svc.GenerateSelfSignedCert(req.Domain, req.ValidDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *Handler) GetCertificateInfo(c *gin.Context) {
	certDir := filepath.Dir(c.Request.URL.Query().Get("path"))
	if certDir == "." {
		// Use default cert dir
	}
	certPath := filepath.Join(h.svc.certDir, "cert.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}
	info, err := h.svc.GetCertificateInfo(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *Handler) UploadCertificate(c *gin.Context) {
	certFile, _, err := c.Request.FormFile("cert")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cert file required"})
		return
	}
	defer certFile.Close()

	keyFile, _, err := c.Request.FormFile("key")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key file required"})
		return
	}
	defer keyFile.Close()

	certData, _ := io.ReadAll(certFile)
	keyData, _ := io.ReadAll(keyFile)

	certPath := filepath.Join(h.svc.certDir, "cert.pem")
	if err := os.WriteFile(certPath, certData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save cert"})
		return
	}
	keyPath := filepath.Join(h.svc.certDir, "key.pem")
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save key"})
		return
	}

	info, err := h.svc.GetCertificateInfo(certPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Files saved but could not read info"})
		return
	}
	c.JSON(http.StatusOK, info)
}
```

```go
// server/internal/certificate/register.go
package certificate

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/certificate", h.GenerateSelfSignedCert)
	rg.GET("/certificate", h.GetCertificateInfo)
	rg.POST("/certificate/upload", h.UploadCertificate)
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/certificate/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add server/internal/certificate/
git commit -m "feat(internal): add certificate slice with self-signed cert generation"
```

### Task 2.2: Create `internal/wireguard/`

**Files:**
- Create: `server/internal/wireguard/service.go`
- Create: `server/internal/wireguard/models.go`
- Create: `server/internal/wireguard/handler.go`
- Create: `server/internal/wireguard/register.go`
- Test: `server/internal/wireguard/service_test.go`
- Test: `server/internal/wireguard/handler_test.go`

- [ ] **Step 1: Write service tests**

```go
// server/internal/wireguard/service_test.go
package wireguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePrivateKey(t *testing.T) {
	svc := NewService(t.TempDir())
	key, err := svc.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GeneratePrivateKey() error = %v", err)
	}
	if len(key) != 44 { // base64 encoded 32 bytes
		t.Errorf("key length = %d, want 44", len(key))
	}
}

func TestGeneratePublicKey(t *testing.T) {
	svc := NewService(t.TempDir())
	priv, _ := svc.GeneratePrivateKey()
	pub, err := svc.GeneratePublicKey(priv)
	if err != nil {
		t.Fatalf("GeneratePublicKey() error = %v", err)
	}
	if len(pub) != 44 {
		t.Errorf("public key length = %d, want 44", len(pub))
	}
}

func TestGeneratePublicKey_invalidBase64(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GeneratePublicKey("invalid-base64!")
	if err == nil {
		t.Error("GeneratePublicKey() expected error for invalid base64")
	}
}

func TestGenerateWireGuardKeysWithCache(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// First call should generate new keys
	resp, err := svc.GenerateWireGuardKeysWithCache("10.0.0.1")
	if err != nil {
		t.Fatalf("GenerateWireGuardKeysWithCache() error = %v", err)
	}
	if resp.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want %q", resp.IP, "10.0.0.1")
	}
	if resp.PrivateKey == "" || resp.PublicKey == "" {
		t.Error("PrivateKey or PublicKey is empty")
	}

	// Second call with same IP should return cached keys
	resp2, err := svc.GenerateWireGuardKeysWithCache("10.0.0.1")
	if err != nil {
		t.Fatalf("GenerateWireGuardKeysWithCache() error = %v", err)
	}
	if resp2.PrivateKey != resp.PrivateKey {
		t.Error("Second call returned different private key (cache miss)")
	}
	if resp2.PublicKey != resp.PublicKey {
		t.Error("Second call returned different public key (cache miss)")
	}
}

func TestGenerateWireGuardKeysWithCache_differentIP(t *testing.T) {
	svc := NewService(t.TempDir())
	resp1, _ := svc.GenerateWireGuardKeysWithCache("10.0.0.1")
	resp2, _ := svc.GenerateWireGuardKeysWithCache("10.0.0.2")
	if resp1.PrivateKey == resp2.PrivateKey {
		t.Error("Different IPs should have different keys")
	}
}

func TestGenerateWireGuardKeysWithCache_noIP(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GenerateWireGuardKeysWithCache("")
	if err == nil {
		t.Error("GenerateWireGuardKeysWithCache() expected error for empty IP")
	}
}

func TestGetKeysCache_empty(t *testing.T) {
	svc := NewService(t.TempDir())
	cache, err := svc.GetKeysCache()
	if err != nil {
		t.Fatalf("GetKeysCache() error = %v", err)
	}
	if len(cache) != 0 {
		t.Errorf("cache length = %d, want 0", len(cache))
	}
}

func TestSaveAndListClientConfigFiles(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	err := svc.SaveClientConfigFile(0, "[Interface]\nPrivateKey = test\n")
	if err != nil {
		t.Fatalf("SaveClientConfigFile() error = %v", err)
	}

	files, err := svc.ListClientConfigFiles()
	if err != nil {
		t.Fatalf("ListClientConfigFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("files count = %d, want 1", len(files))
	}
	if files[0].Name != "client0.conf" {
		t.Errorf("file name = %q, want %q", files[0].Name, "client0.conf")
	}
}

func TestSaveClientConfig(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	err := svc.SaveClientConfig([]byte(`{"key": "value"}`))
	if err != nil {
		t.Fatalf("SaveClientConfig() error = %v", err)
	}

	data, err := svc.GetClientConfig()
	if err != nil {
		t.Fatalf("GetClientConfig() error = %v", err)
	}
	if string(data) != `{"key": "value"}` {
		t.Errorf("config data = %q, want %q", string(data), `{"key": "value"}`)
	}
}

func TestGetClientConfig_notFound(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GetClientConfig()
	if err == nil {
		t.Error("GetClientConfig() expected error for missing file")
	}
}
```

- [ ] **Step 2: Create wireguard package**

```go
// server/internal/wireguard/service.go
package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

type Service struct {
	baseDir string
}

func NewService(baseDir string) *Service {
	return &Service{baseDir: baseDir}
}

func (s *Service) GeneratePrivateKey() (string, error) {
	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", err
	}
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64
	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

func (s *Service) GeneratePublicKey(privateKeyStr string) (string, error) {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}
	if len(privateKey) != 32 {
		return "", fmt.Errorf("invalid private key length: expected 32 bytes, got %d", len(privateKey))
	}
	var privKeyArr [32]byte
	copy(privKeyArr[:], privateKey)
	pubKey, err := curve25519.X25519(privKeyArr[:], curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("failed to generate public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(pubKey), nil
}

func (s *Service) GenerateWireGuardKeysWithCache(ip string) (*KeyCacheResponse, error) {
	if ip == "" {
		return nil, fmt.Errorf("IP address is required")
	}

	cache, err := s.loadKeysCache()
	if err != nil {
		return nil, fmt.Errorf("failed to load keys cache: %w", err)
	}
	for _, entry := range cache {
		if entry.IP == ip {
			return &KeyCacheResponse{
				IP:         entry.IP,
				PrivateKey: entry.PrivateKey,
				PublicKey:  entry.PublicKey,
			}, nil
		}
	}

	privKey, err := s.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	pubKey, err := s.GeneratePublicKey(privKey)
	if err != nil {
		return nil, err
	}

	cache = append(cache, KeyCacheEntry{
		IP:         ip,
		PublicKey:  pubKey,
		PrivateKey: privKey,
	})
	if err := s.saveKeysCache(cache); err != nil {
		return nil, err
	}

	return &KeyCacheResponse{
		IP:         ip,
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}, nil
}

func (s *Service) GetKeysCache() ([]KeyCacheEntry, error) {
	return s.loadKeysCache()
}

func (s *Service) getKeysCacheFilePath() string {
	return filepath.Join(s.baseDir, "wireguard_keys_cache.txt")
}

func (s *Service) loadKeysCache() ([]KeyCacheEntry, error) {
	filePath := s.getKeysCacheFilePath()
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []KeyCacheEntry{}, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var cache []KeyCacheEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 3 {
			continue
		}
		cache = append(cache, KeyCacheEntry{
			IP: parts[0], PublicKey: parts[1], PrivateKey: parts[2],
		})
	}
	return cache, nil
}

func (s *Service) saveKeysCache(cache []KeyCacheEntry) error {
	var lines []string
	for _, entry := range cache {
		lines = append(lines, fmt.Sprintf("%s %s %s", entry.IP, entry.PublicKey, entry.PrivateKey))
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(s.getKeysCacheFilePath(), []byte(content), 0644)
}

func (s *Service) GetPublicIP() (string, error) {
	sources := []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://icanhazip.com",
		"https://checkip.amazonaws.com",
	}
	timeout := 5 * time.Second
	var lastErr error
	for _, source := range sources {
		ip, err := s.fetchIPFromSource(source, timeout)
		if err != nil {
			lastErr = err
			continue
		}
		return ip, nil
	}
	if lastErr != nil {
		return "", fmt.Errorf("all IP sources failed: %w", lastErr)
	}
	return "", fmt.Errorf("no IP sources available")
}

func (s *Service) fetchIPFromSource(url string, timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP: %s", ip)
	}
	return ip, nil
}

func (s *Service) SaveClientConfig(configData []byte) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.baseDir, "client-config.json"), configData, 0644)
}

func (s *Service) GetClientConfig() ([]byte, error) {
	path := filepath.Join(s.baseDir, "client-config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("client config not found")
	}
	return os.ReadFile(path)
}

func (s *Service) SaveClientConfigFile(clientIndex int, configContent string) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	confPath := filepath.Join(s.baseDir, fmt.Sprintf("client%d.conf", clientIndex))
	return os.WriteFile(confPath, []byte(configContent), 0644)
}

func (s *Service) ListClientConfigFiles() ([]ClientConfigFile, error) {
	if _, err := os.Stat(s.baseDir); os.IsNotExist(err) {
		return []ClientConfigFile{}, nil
	}
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, err
	}
	var configs []ClientConfigFile
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".conf" {
			content, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
			if err != nil {
				continue
			}
			configs = append(configs, ClientConfigFile{
				Name: entry.Name(), Content: string(content),
			})
		}
	}
	return configs, nil
}
```

```go
// server/internal/wireguard/models.go
package wireguard

type WireGuardKeyPair struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type KeyCacheEntry struct {
	IP         string `json:"ip"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

type KeyCacheResponse struct {
	IP         string `json:"ip"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type ClientConfigFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/wireguard/ -v`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add server/internal/wireguard/
git commit -m "feat(internal): add wireguard slice with key generation and client config"
```

### Task 2.3: Create `internal/warp/`

**Files:**
- Create: `server/internal/warp/service.go`
- Create: `server/internal/warp/models.go`
- Create: `server/internal/warp/scanner.go`
- Create: `server/internal/warp/handler.go`
- Create: `server/internal/warp/register.go`
- Test: `server/internal/warp/service_test.go`
- Test: `server/internal/warp/scanner_test.go`
- Test: `server/internal/warp/handler_test.go`

The scanner tests already exist in `services/warp_scanner_test.go` — adapt them. For service tests, mock the HTTP calls to Cloudflare API.

- [ ] **Step 1: Write service tests**

```go
// server/internal/warp/service_test.go
package warp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRegisterAndBuildOutbound(t *testing.T) {
	// Mock Cloudflare API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/v0a2158/reg" {
			json.NewEncoder(w).Encode(WarpRegisterResponse{
				ID:    "test-device-id",
				Token: "test-token",
				Account: WarpAccount{
					AccountType: "free",
					WarpPlus:    false,
				},
				Config: WarpConfig{
					ClientID: "AAAA",
					Interface: WarpInterface{
						Addresses: WarpInterfaceAddr{
							V4: "172.16.0.2",
							V6: "fd01::2",
						},
					},
					Peers: []WarpPeer{
						{
							PublicKey: "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
							Endpoint: WarpPeerEndpoint{
								Host: "engage.cloudflareclient.com",
							},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Override API base URL for testing
	warpAPIBase = server.URL

	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	rec, err := svc.RegisterDevice()
	if err != nil {
		t.Fatalf("RegisterDevice() error = %v", err)
	}
	if rec.Device.ID != "test-device-id" {
		t.Errorf("Device.ID = %q, want %q", rec.Device.ID, "test-device-id")
	}
	if rec.PrivateKey == "" || rec.PublicKey == "" {
		t.Error("PrivateKey or PublicKey is empty")
	}

	// Build outbound
	outbound, err := svc.BuildWarpOutbound("", 0, 0)
	if err != nil {
		t.Fatalf("BuildWarpOutbound() error = %v", err)
	}
	if outbound["type"] != "wireguard" {
		t.Errorf("outbound type = %q, want %q", outbound["type"], "wireguard")
	}
	if outbound["tag"] != "proxy_out" {
		t.Errorf("outbound tag = %q, want %q", outbound["tag"], "proxy_out")
	}

	// Verify file was saved
	if _, err := os.Stat(filepath.Join(tmpDir, "warp-account.json")); os.IsNotExist(err) {
		t.Error("warp-account.json not created")
	}
}

func RegisterDevice_noServer(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	_, err := svc.RegisterDevice()
	if err == nil {
		t.Error("RegisterDevice() expected error with no server")
	}
}
```

```go
// server/internal/warp/service.go
package warp

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/curve25519"
)

var warpAPIBase = "https://api.cloudflareclient.com"
const (
	warpAPIVersion  = "v0a2158"
	warpClientUA    = "okhttp/3.12.1"
	warpClientVer   = "a-6.11-2158"
	warpDefaultHost = "engage.cloudflareclient.com"
	warpDefaultPort = 2408
)

type Service struct {
	baseDir string
	record  *WarpRecord
}

func NewService(baseDir string) *Service {
	return &Service{baseDir: baseDir}
}

func (s *Service) RegisterDevice() (*WarpRecord, error) {
	privKey, err := generatePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	pubKey, err := generatePublicKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("generate public key: %w", err)
	}

	serial, err := randomHexStr(8)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"key": pubKey, "install_id": "", "fcm_token": "",
		"tos": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"model": "PC", "serial_number": serial, "locale": "en_US",
	}
	payload, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/%s/reg", warpAPIBase, warpAPIVersion)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", warpClientUA)
	req.Header.Set("CF-Client-Version", warpClientVer)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("WARP registration failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("WARP registration HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var regResp WarpRegisterResponse
	if err := json.Unmarshal(respBody, &regResp); err != nil {
		return nil, fmt.Errorf("parse WARP response: %w", err)
	}
	if len(regResp.Config.Peers) == 0 {
		return nil, fmt.Errorf("WARP response missing peer config")
	}

	now := time.Now().Format(time.RFC3339)
	s.record = &WarpRecord{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Device:     regResp,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.saveRecord(); err != nil {
		return nil, err
	}
	return s.record, nil
}

func (s *Service) LoadRecord() (*WarpRecord, error) {
	path := s.recordPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec WarpRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	s.record = &rec
	return &rec, nil
}

func (s *Service) DeleteRecord() error {
	s.record = nil
	path := s.recordPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func (s *Service) BindLicense(license string) (*WarpRecord, error) {
	if s.record == nil {
		// Try loading
		if _, err := s.LoadRecord(); err != nil || s.record == nil {
			return nil, fmt.Errorf("no WARP device registered")
		}
	}
	if license == "" {
		return nil, fmt.Errorf("license cannot be empty")
	}

	rec := s.record
	body, _ := json.Marshal(map[string]string{"license": license})
	url := fmt.Sprintf("%s/%s/reg/%s/account", warpAPIBase, warpAPIVersion, rec.Device.ID)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("User-Agent", warpClientUA)
	req.Header.Set("CF-Client-Version", warpClientVer)
	req.Header.Set("Authorization", "Bearer "+rec.Device.Token)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("license request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("license HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var acct WarpAccount
	if err := json.Unmarshal(respBody, &acct); err != nil {
		return nil, fmt.Errorf("parse license response: %w", err)
	}

	rec.Device.Account.License = acct.License
	rec.Device.Account.AccountType = acct.AccountType
	rec.Device.Account.WarpPlus = acct.WarpPlus
	if acct.ID != "" {
		rec.Device.Account.ID = acct.ID
	}
	if acct.PremiumData > 0 {
		rec.Device.Account.PremiumData = acct.PremiumData
	}
	rec.UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.saveRecord(); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Service) BuildWarpOutbound(endpointHost string, endpointPort int, mtu int) (map[string]interface{}, error) {
	if s.record == nil {
		if _, err := s.LoadRecord(); err != nil || s.record == nil {
			return nil, fmt.Errorf("no WARP device record")
		}
	}
	rec := s.record
	if len(rec.Device.Config.Peers) == 0 {
		return nil, fmt.Errorf("WARP record missing peer config")
	}

	var reserved []int
	if rec.Device.Config.ClientID != "" {
		raw, err := decodeWarpClientID(rec.Device.Config.ClientID)
		if err != nil {
			return nil, fmt.Errorf("parse client_id: %w", err)
		}
		reserved = []int{int(raw[0]), int(raw[1]), int(raw[2])}
	}

	host := endpointHost
	if host == "" {
		host = warpDefaultHost
	}
	port := endpointPort
	if port == 0 {
		port = warpDefaultPort
	}
	if mtu <= 0 || mtu > 1500 {
		mtu = 1280
	}

	v4 := rec.Device.Config.Interface.Addresses.V4
	v6 := rec.Device.Config.Interface.Addresses.V6
	var addresses []string
	if v4 != "" {
		if !strings.Contains(v4, "/") {
			v4 += "/32"
		}
		addresses = append(addresses, v4)
	}
	if v6 != "" {
		if !strings.Contains(v6, "/") {
			v6 += "/128"
		}
		addresses = append(addresses, v6)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("WARP record missing client address")
	}

	peer := rec.Device.Config.Peers[0]
	peerMap := map[string]interface{}{
		"address": host, "port": port, "public_key": peer.PublicKey,
		"allowed_ips": []string{"0.0.0.0/0", "::/0"},
	}
	if len(reserved) == 3 {
		peerMap["reserved"] = reserved
	}

	return map[string]interface{}{
		"type": "wireguard", "tag": "proxy_out",
		"address": addresses, "private_key": rec.PrivateKey,
		"mtu": mtu, "peers": []interface{}{peerMap},
	}, nil
}

func (s *Service) recordPath() string {
	return filepath.Join(s.baseDir, "warp-account.json")
}

func (s *Service) saveRecord() error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.record, "", "  ")
	if err != nil {
		return err
	}
	path := s.recordPath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	_ = os.Chmod(tmp, 0600)
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Chmod(path, 0600)
	return nil
}

func generatePrivateKey() (string, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

func generatePublicKey(priv string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(priv)
	if err != nil {
		return "", err
	}
	var arr [32]byte
	copy(arr[:], b)
	pub, err := curve25519.X25519(arr[:], curve25519.Basepoint)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

func randomHexStr(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func decodeWarpClientID(cid string) ([]byte, error) {
	if cid == "" {
		return nil, fmt.Errorf("empty client_id")
	}
	encs := []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	}
	for _, enc := range encs {
		if b, err := enc.DecodeString(cid); err == nil && len(b) >= 3 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("invalid client_id base64")
}
```

- [ ] **Step 4: Verify scanner tests are ported**

Create `server/internal/warp/scanner_test.go` by adapting `services/warp_scanner_test.go` — same test logic, new package path.

Run: `go test ./internal/warp/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/warp/
git commit -m "feat(internal): add warp slice with device registration, license, and endpoint scanner"
```

---

## Step 3: `internal/subscription/`

### Task 3.1: Write parser tests (pure logic, no mocks)

**Files:**
- Create: `server/internal/subscription/parser_vmess.go`
- Create: `server/internal/subscription/parser_vless.go`
- Create: `server/internal/subscription/parser_trojan.go`
- Create: `server/internal/subscription/parser_ss.go`
- Create: `server/internal/subscription/parser_clash.go`
- Test: `server/internal/subscription/parser_vmess_test.go`
- Test: `server/internal/subscription/parser_vless_test.go`
- Test: `server/internal/subscription/parser_trojan_test.go`
- Test: `server/internal/subscription/parser_ss_test.go`
- Test: `server/internal/subscription/parser_clash_test.go`

- [ ] **Step 1: Write VMess parser tests**

```go
// server/internal/subscription/parser_vmess_test.go
package subscription

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestParseVMessNode_standard(t *testing.T) {
	vmessObj := map[string]string{
		"v": "2", "ps": "Test Node", "add": "1.2.3.4", "port": "443",
		"id": "uuid-here", "aid": "0", "net": "tcp", "type": "none",
		"host": "", "path": "", "tls": "",
	}
	data, _ := json.Marshal(vmessObj)
	link := "vmess://" + base64.StdEncoding.EncodeToString(data)

	node, err := parseVMessNode(link)
	if err != nil {
		t.Fatalf("parseVMessNode() error = %v", err)
	}
	if node.Name != "Test Node" {
		t.Errorf("Name = %q, want %q", node.Name, "Test Node")
	}
	if node.Protocol != "vmess" {
		t.Errorf("Protocol = %q, want %q", node.Protocol, "vmess")
	}
	if node.Address != "1.2.3.4" {
		t.Errorf("Address = %q, want %q", node.Address, "1.2.3.4")
	}
	if node.Port != 443 {
		t.Errorf("Port = %d, want %d", node.Port, 443)
	}
}

func TestParseVMessNode_withTLS(t *testing.T) {
	vmessObj := map[string]string{
		"v": "2", "ps": "TLS Node", "add": "example.com", "port": "8443",
		"id": "uuid", "aid": "0", "net": "tcp", "type": "none",
		"host": "", "path": "", "tls": "tls", "sni": "example.com",
	}
	data, _ := json.Marshal(vmessObj)
	link := "vmess://" + base64.StdEncoding.EncodeToString(data)

	node, err := parseVMessNode(link)
	if err != nil {
		t.Fatalf("parseVMessNode() error = %v", err)
	}
	tls, ok := node.Outbound["tls"].(map[string]interface{})
	if !ok {
		t.Fatal("Outbound missing tls field")
	}
	if tls["enabled"] != true {
		t.Error("TLS not enabled")
	}
}

func TestParseVMessNode_withWS(t *testing.T) {
	vmessObj := map[string]string{
		"v": "2", "ps": "WS Node", "add": "example.com", "port": "443",
		"id": "uuid", "aid": "0", "net": "ws", "type": "none",
		"host": "example.com", "path": "/ws", "tls": "",
	}
	data, _ := json.Marshal(vmessObj)
	link := "vmess://" + base64.StdEncoding.EncodeToString(data)

	node, err := parseVMessNode(link)
	if err != nil {
		t.Fatalf("parseVMessNode() error = %v", err)
	}
	transport, ok := node.Outbound["transport"].(map[string]interface{})
	if !ok {
		t.Fatal("Outbound missing transport field")
	}
	if transport["type"] != "ws" {
		t.Errorf("transport type = %q, want %q", transport["type"], "ws")
	}
}

func TestParseVMessNode_invalidBase64(t *testing.T) {
	_, err := parseVMessNode("vmess://not-base64!!!")
	if err == nil {
		t.Error("parseVMessNode() expected error for invalid base64")
	}
}
```

- [ ] **Step 2: Write VLESS parser tests**

```go
// server/internal/subscription/parser_vless_test.go
package subscription

import "testing"

func TestParseVLESSNode_standard(t *testing.T) {
	link := "vless://uuid@example.com:443?type=tcp&security=tls&sni=example.com&fp=chrome#MyVLESS"
	node, err := parseVLESSNode(link)
	if err != nil {
		t.Fatalf("parseVLESSNode() error = %v", err)
	}
	if node.Name != "MyVLESS" {
		t.Errorf("Name = %q, want %q", node.Name, "MyVLESS")
	}
	if node.Protocol != "vless" {
		t.Errorf("Protocol = %q, want %q", node.Protocol, "vless")
	}
	if node.Address != "example.com" {
		t.Errorf("Address = %q, want %q", node.Address, "example.com")
	}
	if node.Port != 443 {
		t.Errorf("Port = %d, want %d", node.Port, 443)
	}
}

func TestParseVLESSNode_reality(t *testing.T) {
	link := "vless://uuid@example.com:443?type=tcp&security=reality&sni=www.example.com&pbk=publickey&sid=1234&fp=chrome&flow=xtls-rprx-vision#RealityNode"
	node, err := parseVLESSNode(link)
	if err != nil {
		t.Fatalf("parseVLESSNode() error = %v", err)
	}
	if node.Outbound["flow"] != "xtls-rprx-vision" {
		t.Errorf("flow = %q, want %q", node.Outbound["flow"], "xtls-rprx-vision")
	}
	tls := node.Outbound["tls"].(map[string]interface{})
	reality := tls["reality"].(map[string]interface{})
	if reality["public_key"] != "publickey" {
		t.Errorf("reality public_key = %q, want %q", reality["public_key"], "publickey")
	}
}

func TestParseVLESSNode_malformed(t *testing.T) {
	tests := []string{
		"vless://invalid",
		"vless://@",
		"vless://uuid@@host:443",
	}
	for _, link := range tests {
		_, err := parseVLESSNode(link)
		if err == nil {
			t.Errorf("parseVLESSNode(%q) expected error", link)
		}
	}
}
```

- [ ] **Step 3: Write Trojan parser tests**

```go
// server/internal/subscription/parser_trojan_test.go
package subscription

import "testing"

func TestParseTrojanNode_standard(t *testing.T) {
	link := "trojan://password@example.com:443?type=tcp&sni=example.com#TrojanNode"
	node, err := parseTrojanNode(link)
	if err != nil {
		t.Fatalf("parseTrojanNode() error = %v", err)
	}
	if node.Name != "TrojanNode" {
		t.Errorf("Name = %q, want %q", node.Name, "TrojanNode")
	}
	if node.Protocol != "trojan" {
		t.Errorf("Protocol = %q, want %q", node.Protocol, "trojan")
	}
	if node.Address != "example.com" {
		t.Errorf("Address = %q, want %q", node.Address, "example.com")
	}
	if node.Port != 443 {
		t.Errorf("Port = %d, want %d", node.Port, 443)
	}
}

func TestParseTrojanNode_withWS(t *testing.T) {
	link := "trojan://pass@example.com:443?type=ws&path=/ws&host=example.com#WSNode"
	node, err := parseTrojanNode(link)
	if err != nil {
		t.Fatalf("parseTrojanNode() error = %v", err)
	}
	transport := node.Outbound["transport"].(map[string]interface{})
	if transport["type"] != "ws" {
		t.Errorf("transport type = %q, want %q", transport["type"], "ws")
	}
}
```

- [ ] **Step 4: Write Shadowsocks parser tests**

```go
// server/internal/subscription/parser_ss_test.go
package subscription

import "testing"

func TestParseShadowsocksNode_sip002(t *testing.T) {
	// SIP002 format: ss://BASE64(method:password)@server:port#name
	link := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@example.com:8388#SSNode"
	node, err := parseShadowsocksNode(link)
	if err != nil {
		t.Fatalf("parseShadowsocksNode() error = %v", err)
	}
	if node.Name != "SSNode" {
		t.Errorf("Name = %q, want %q", node.Name, "SSNode")
	}
	if node.Protocol != "shadowsocks" {
		t.Errorf("Protocol = %q, want %q", node.Protocol, "shadowsocks")
	}
	if node.Address != "example.com" {
		t.Errorf("Address = %q, want %q", node.Address, "example.com")
	}
	if node.Port != 8388 {
		t.Errorf("Port = %d, want %d", node.Port, 8388)
	}
}

func TestParseShadowsocksNode_legacy(t *testing.T) {
	// Legacy format: ss://BASE64(method:password@server:port)#name
	link := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmRAZXhhbXBsZS5jb206ODM4OA==#LegacySS"
	node, err := parseShadowsocksNode(link)
	if err != nil {
		t.Fatalf("parseShadowsocksNode() error = %v", err)
	}
	if node.Name != "LegacySS" {
		t.Errorf("Name = %q, want %q", node.Name, "LegacySS")
	}
}
```

- [ ] **Step 5: Write Clash YAML parser tests**

```go
// server/internal/subscription/parser_clash_test.go
package subscription

import "testing"

func TestParseClashYAML_standard(t *testing.T) {
	yaml := `proxies:
  - name: "Clash VMess"
    type: vmess
    server: 1.2.3.4
    port: 443
    uuid: "uuid-here"
    alterId: 0
    cipher: auto
    udp: true
  - name: "Clash Trojan"
    type: trojan
    server: trojan.example.com
    port: 443
    password: "secret"
    sni: trojan.example.com
`
	nodes, err := parseClashYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("parseClashYAML() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes count = %d, want 2", len(nodes))
	}
	if nodes[0].Protocol != "vmess" {
		t.Errorf("nodes[0].Protocol = %q, want %q", nodes[0].Protocol, "vmess")
	}
	if nodes[1].Protocol != "trojan" {
		t.Errorf("nodes[1].Protocol = %q, want %q", nodes[1].Protocol, "trojan")
	}
}

func TestParseClashYAML_empty(t *testing.T) {
	_, err := parseClashYAML([]byte("proxies: []"))
	if err != nil {
		t.Fatalf("parseClashYAML() error = %v", err)
	}
}

func TestDetectClashYAML(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"proxies:\n  - name: test", false},
		{"proxies:\n  - name: test\nproxy-groups:\n  - name: Proxy", true},
		{"mixed-port: 7890\nproxies:\n  - name: test", true},
		{"proxies:\n  - name: test\ntype: vmess\nserver: 1.2.3.4\nport: 443", false},
	}
	for _, tt := range tests {
		got := isClashYAML(tt.content)
		if got != tt.want {
			t.Errorf("isClashYAML(%q) = %v, want %v", tt.content[:20]+"...", got, tt.want)
		}
	}
}
```

- [ ] **Step 6: Create parser implementations (port from services/subscription.go)**

The parser functions are pure translations of the existing code from `services/subscription.go` into the new package. Each function stays the same — it takes a string link, returns `(*types.ProxyNode, error)`.

- [ ] **Step 7: Run all parser tests**

Run: `go test ./internal/subscription/ -run TestParse -v`
Expected: ALL PASS

- [ ] **Step 8: Create service + store with tests**

Test `AddSubscription`, `UpdateSubscription`, `DeleteSubscription`, `GetAllNodes`, `RefreshAllSubscriptions`, `SaveProbeResults`, `SaveSpeedTestResults` using mock store and mock HTTP fetcher.

- [ ] **Step 9: Commit**

```bash
git add server/internal/subscription/
git commit -m "feat(internal): add subscription slice with protocol parsers and CRUD"
```

---

## Steps 4-6: Remaining Domain Slices

### Task 4.1: Create `internal/singbox/`

**Files:**
- Create: `server/internal/singbox/service.go` — `SaveConfig`, `GetConfig`, `RunContainer`, `StopContainer`, `ContainerStatus`, `ContainerLogs`, `EnsureImage`, `GetVersion`
- Create: `server/internal/singbox/models.go` — `NamedConfigInfo`
- Create: `server/internal/singbox/interfaces.go` — `ContainerManager` interface
- Create: `server/internal/singbox/handler.go` — 20 HTTP handlers
- Create: `server/internal/singbox/register.go` — `RegisterRoutes()`
- Test: `server/internal/singbox/service_test.go`

**Service interface:**
```go
type Service struct {
	docker docker.ContainerAPI
	cfg    *config.Config
}

func NewService(docker docker.ContainerAPI, cfg *config.Config) *Service
func (s *Service) SaveConfig(data []byte) (string, error)
func (s *Service) GetConfig() ([]byte, error)
func (s *Service) RunContainer() (string, error)
func (s *Service) StopContainer() error
func (s *Service) ContainerStatus() (running bool, containerID string)
func (s *Service) ContainerLogs() string
func (s *Service) EnsureImage() error
func (s *Service) GetVersion() (string, error)
func (s *Service) SaveNamedConfig(name string, data []byte) error
func (s *Service) LoadNamedConfig(name string) ([]byte, error)
func (s *Service) DeleteNamedConfig(name string) error
func (s *Service) ListNamedConfigs() ([]NamedConfigInfo, error)
func (s *Service) RunNamedContainer(name string) (string, error)
func (s *Service) StopNamedContainer(name string) error
func (s *Service) NamedContainerStatus(name string) (running bool, containerID string)
func (s *Service) NamedContainerLogs(name string) string
func (s *Service) CheckNamedConfig(name string) (valid bool, output string, error)
func (s *Service) ListAllContainers() ([]docker.ContainerInfo, error)
```

**Test patterns:**
```go
func TestSaveAndGetConfig(t *testing.T) {
    svc := NewService(docker.NewMockClient(), config.TestConfig(t.TempDir()))
    _, err := svc.SaveConfig([]byte(`{"log":{"level":"info"}}`))
    assert.NoError(t, err)
    data, err := svc.GetConfig()
    assert.NoError(t, err)
    assert.JSONEq(t, `{"log":{"level":"info"}}`, string(data))
}

func TestGetConfig_notFound(t *testing.T) {
    svc := NewService(docker.NewMockClient(), config.TestConfig(t.TempDir()))
    _, err := svc.GetConfig()
    assert.Error(t, err) // file doesn't exist yet
}
```

**Docker mock (in `internal/pkg/docker/mock_test.go`):**
```go
// Docker mock for testing
type MockContainerAPI struct {
    CreateContainerFn func(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error)
    StartContainerFn  func(ctx context.Context, id string) error
    // ... all interface methods
}
func NewMockClient() *MockContainerAPI {
    return &MockContainerAPI{
        CreateContainerFn: func(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error) {
            return "mock-id", nil
        },
        StartContainerFn: func(ctx context.Context, id string) error { return nil },
        // ... sensible defaults
    }
}
```

- [ ] **Step 1-5: TDD cycle for each public method** — write test, see it fail, implement, see it pass, commit.

### Task 5.1: Create `internal/prober/`

**Files:**
- Create: `server/internal/prober/engine.go` — `Prober` struct (goroutine loop, semaphore, ring buffer)
- Create: `server/internal/prober/service.go` — init, CRUD, file persistence
- Create: `server/internal/prober/models.go` — `ProberConfig` (private)
- Create: `server/internal/prober/handler.go` — 13 HTTP handlers
- Create: `server/internal/prober/register.go`
- Test: `server/internal/prober/engine_test.go`
- Test: `server/internal/prober/service_test.go`

**Engine public API:**
```go
type Prober struct { /* unexported fields */ }
func NewProber(config ProberConfig) *Prober
func (p *Prober) Start()
func (p *Prober) Stop()
func (p *Prober) IsRunning() bool
func (p *Prober) AddNode(node types.ProbeNode)
func (p *Prober) RemoveNode(tag string)
func (p *Prober) ClearNodes()
func (p *Prober) UpdateNodes(nodes []types.ProbeNode)
func (p *Prober) GetResult(tag string) *types.ProbeResult
func (p *Prober) GetAllResults() map[string]*types.ProbeResult
func (p *Prober) GetBestNode() *types.ProbeResult
func (p *Prober) GetOnlineNodes() []*types.ProbeResult
func (p *Prober) GetStats() map[string]interface{}
func (p *Prober) SaveNodesToFile() error
func (p *Prober) LoadNodesFromFile() error
```

**Test patterns (port from `services/prober_test.go`):**
```go
func TestProberAddRemoveNode(t *testing.T) {
    p := NewProber(DefaultProberConfig())
    p.AddNode(types.ProbeNode{Tag: "test", Protocol: "vmess", Address: "127.0.0.1", Port: 10086})
    results := p.GetAllResults()
    if len(results) != 1 { t.Errorf("got %d results, want 1", len(results)) }
    p.RemoveNode("test")
    results = p.GetAllResults()
    if len(results) != 0 { t.Errorf("got %d results, want 0", len(results)) }
}

func TestProberStartStop(t *testing.T) {
    p := NewProber(DefaultProberConfig())
    p.Start()
    if !p.IsRunning() { t.Error("Prober should be running after Start()") }
    p.Stop()
    if p.IsRunning() { t.Error("Prober should not be running after Stop()") }
}

func TestProberEngine_ContextCancellation(t *testing.T) {
    // Port from TestProberContextCancellation in services/prober_test.go
}
```

- [ ] **Step 4: Port all existing prober tests** from `services/prober_test.go` to `internal/prober/engine_test.go`
- [ ] **Step 5: Run tests** — `go test ./internal/prober/ -v` — ALL PASS
- [ ] **Step 6: Commit**

### Task 5.2: Create `internal/speedtest/`

**Files:**
- Create: `server/internal/speedtest/service.go`
- Create: `server/internal/speedtest/models.go`
- Create: `server/internal/speedtest/handler.go`
- Create: `server/internal/speedtest/register.go`
- Test: `server/internal/speedtest/service_test.go`

**Public API:**
```go
type Service struct { /* ... */ }
func NewService(docker docker.ContainerAPI, cfg *config.Config) *Service
func (s *Service) StartSpeedTest(nodeProvider types.NodeProvider) error
func (s *Service) GetSpeedTestState() *SpeedTestState
func (s *Service) StopSpeedTest()
```

**Test patterns:**
```go
func TestPickFreePort(t *testing.T) {
    port, err := pickFreePort()
    assert.NoError(t, err)
    assert.True(t, port > 0)
}

func TestBuildSpeedTestConfig(t *testing.T) {
    node := types.ProxyNode{
        Outbound: map[string]interface{}{
            "type": "vless", "server": "example.com", "server_port": 443,
        },
    }
    cfg := buildSpeedTestConfig(node, "test-tag", 10800)
    assert.Equal(t, "vless", cfg["outbounds"].([]interface{})[0].(map[string]interface{})["type"])
}
```

- [ ] **Step 1-5: TDD cycle**
- [ ] **Step 6: Commit**

### Task 6.1: Create `internal/scheduler/`

**Files:**
- Create: `server/internal/scheduler/service.go`
- Create: `server/internal/scheduler/interfaces.go`
- Test: `server/internal/scheduler/service_test.go`

**Public API:**
```go
type SubscriptionUpdater interface {
    LoadAll() ([]types.SubscriptionEntry, error)
    UpdateOne(id string) (*types.SubscriptionEntry, error)
}

type ContainerManager interface {
    UpdateAndRestart(name string, configData []byte) error
    Status(name string) (running bool, containerID string)
}

func New(subUpdater SubscriptionUpdater, containerMgr ContainerManager) *Scheduler
func (s *Scheduler) Start()
func (s *Scheduler) Stop()
```

**Test patterns (full mock of both interfaces):**
```go
type mockSubUpdater struct {
    entries []types.SubscriptionEntry
    updated map[string]bool
}
func (m *mockSubUpdater) LoadAll() ([]types.SubscriptionEntry, error) { return m.entries, nil }
func (m *mockSubUpdater) UpdateOne(id string) (*types.SubscriptionEntry, error) {
    m.updated[id] = true
    return &m.entries[0], nil
}

func TestScheduler_autoUpdateTrigger(t *testing.T) {
    subMock := &mockSubUpdater{
        entries: []types.SubscriptionEntry{{
            ID: "test", AutoUpdate: true, UpdateInterval: 1,
            LastUpdated: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
        }},
        updated: make(map[string]bool),
    }
    containerMock := &mockContainerManager{}
    sched := New(subMock, containerMock)
    
    // Call the internal check function directly (not via goroutine)
    sched.checkAndAutoUpdateSubscriptions()
    
    if !subMock.updated["test"] {
        t.Error("Subscription should have been auto-updated")
    }
}

func TestScheduler_skipIfNotDue(t *testing.T) {
    subMock := &mockSubUpdater{
        entries: []types.SubscriptionEntry{{
            ID: "test", AutoUpdate: true, UpdateInterval: 24,
            LastUpdated: time.Now().Format(time.RFC3339), // just updated
        }},
        updated: make(map[string]bool),
    }
    sched := New(subMock, &mockContainerManager{})
    sched.checkAndAutoUpdateSubscriptions()
    if subMock.updated["test"] {
        t.Error("Subscription should NOT be updated if not due")
    }
}

func TestScheduler_skipIfAutoUpdateDisabled(t *testing.T) {
    subMock := &mockSubUpdater{
        entries: []types.SubscriptionEntry{{
            ID: "test", AutoUpdate: false, // disabled
        }},
        updated: make(map[string]bool),
    }
    sched := New(subMock, &mockContainerManager{})
    sched.checkAndAutoUpdateSubscriptions()
    if subMock.updated["test"] {
        t.Error("Subscription should NOT be updated if AutoUpdate is false")
    }
}
```

- [ ] **Step 1-4: Write failing tests → implement → pass → commit**
- [ ] **Step 5: Commit**

```bash
git add server/internal/scheduler/
git commit -m "feat(internal): add scheduler slice with auto-update of subscriptions"
```

---

## Step 7: Rewrite `main.go`

- Wire all dependencies
- Register routes from all slices
- Remove `init.go` calls, replace with explicit construction
- Keep old `handlers/` and `services/` imports temporarily for reference

---

## Step 8: Delete old packages

- Delete `server/handlers/`
- Delete `server/services/`
- Run `go build ./...` — must compile

---

## Step 9-11: Verification

- `go test -race -cover ./...` — all tests pass, race-free, ≥95% coverage
- `golangci-lint run ./...` — clean
- `npm run build` + `go build -o sing-box-ui .` — full binary builds
