# Native Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add NativeRuntime (process-based) alongside existing DockerRuntime for sing-box lifecycle, selected via Go build tags (`openwrt` → native, default → Docker).

**Architecture:** Introduce a `Runtime` interface with two implementations — `DockerRuntime` (existing Docker SDK) and `NativeRuntime` (os/exec + PID files). Similarly, a `TempRuntime` interface for speedtest. The `singbox.Service` and `speedtest.Service` consume these interfaces instead of Docker directly. `main.go` obtains the runtime via a build-tag-guarded factory.

**Tech Stack:** Go 1.24, Docker SDK (conditional), os/exec, syscall (SIGTERM/SIGKILL), gin, build tags

---

## Files Overview

| File | Action | Purpose |
|------|--------|---------|
| `server/internal/singbox/runtime.go` | Create | `Runtime` interface + `InstanceInfo` struct |
| `server/internal/singbox/runtime_docker.go` | Create | `DockerRuntime` + factory (`//go:build !openwrt`) |
| `server/internal/singbox/runtime_native.go` | Create | `NativeRuntime` + factory (`//go:build openwrt`) |
| `server/internal/speedtest/runtimer.go` | Create | `TempRuntime` interface + `InstanceInfo` |
| `server/internal/speedtest/runtimer_docker.go` | Create | `DockerTempRuntime` + factory (`//go:build !openwrt`) |
| `server/internal/speedtest/runtimer_native.go` | Create | `NativeTempRuntime` + factory (`//go:build openwrt`) |
| `server/internal/pkg/config/config.go` | Modify | Add `singboxBinPath` field + getter/setter |
| `server/internal/singbox/service.go` | Modify | Replace `ContainerManager` → `Runtime` |
| `server/internal/singbox/interfaces.go` | Modify | Remove `ContainerManager` (superseded by `Runtime`) |
| `server/internal/speedtest/service.go` | Modify | Replace `ContainerManager` → `TempRuntime` |
| `server/internal/speedtest/interfaces.go` | Modify | Replace `ContainerManager` → `TempRuntime` |
| `server/main.go` | Modify | Remove docker import, use factory functions |
| `Makefile` | Modify | Add OpenWrt cross-compile target |

---

### Task 1: Define Runtime Interface + Config Changes

**Files:**
- Create: `server/internal/singbox/runtime.go`
- Modify: `server/internal/pkg/config/config.go`

- [ ] **Step 1: Write `runtime.go` with the Runtime interface and InstanceInfo struct**

```go
// Package singbox provides services for managing sing-box configuration and containers.
package singbox

import "context"

// Runtime abstracts the lifecycle of a sing-box instance.
// Implementations: DockerRuntime (via Docker SDK), NativeRuntime (via os/exec).
type Runtime interface {
	// Start launches an instance with the given name and config file path.
	// Returns an opaque instance identifier (container ID or "pid:<N>").
	Start(ctx context.Context, name string, configPath string) (id string, err error)

	// Stop terminates an instance gracefully within the optional timeout (seconds).
	// If timeout is nil, a default timeout applies.
	Stop(ctx context.Context, name string, timeout *int) error

	// Status reports whether an instance is running and its identifier.
	Status(ctx context.Context, name string) (running bool, id string, err error)

	// Logs returns recent log lines from an instance.
	// tail specifies the number of lines (empty defaults to 100).
	Logs(ctx context.Context, name string, tail string) (string, error)

	// Version returns the sing-box version string.
	Version(ctx context.Context) (string, error)

	// List returns all instances managed by this runtime.
	List(ctx context.Context) ([]InstanceInfo, error)

	// Close releases any underlying resources (e.g. Docker client).
	Close() error
}

// InstanceInfo describes a sing-box instance.
type InstanceInfo struct {
	Name    string `json:"name"`
	ID      string `json:"container_id,omitempty"`
	Running bool   `json:"running"`
	State   string `json:"state,omitempty"`
}
```

- [ ] **Step 2: Add `singboxBinPath` to config**

Edit `server/internal/pkg/config/config.go`:

Add field to Config struct:
```go
type Config struct {
	dataDir        string
	hostDataDir    string
	listenAddr     string
	singboxDir     string
	singboxBinPath string // new: path to sing-box binary for native runtime
}
```

Add setter and getter methods:
```go
// SetSingboxBinPath sets the path to the sing-box binary.
func (c *Config) SetSingboxBinPath(p string) {
	c.singboxBinPath = p
}

// GetSingboxBinPath returns the path to the sing-box binary.
func (c *Config) GetSingboxBinPath() string {
	return c.singboxBinPath
}
```

- [ ] **Step 3: Run tests to verify nothing is broken**

```bash
cd server && go build ./... && go test ./...
```

Expected: All existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add server/internal/singbox/runtime.go server/internal/pkg/config/config.go
git commit -m "feat: add Runtime interface and singboxBinPath config field"
```

---

### Task 2: Implement DockerRuntime

**Files:**
- Create: `server/internal/singbox/runtime_docker.go`

- [ ] **Step 1: Write DockerRuntime**

```go
// Package singbox provides services for managing sing-box configuration and containers.
//
//go:build !openwrt

package singbox

import (
	"context"
	"fmt"
	"log"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
)

// DockerRuntime manages sing-box instances via Docker containers.
type DockerRuntime struct {
	client docker.ContainerAPI
	cfg    *config.Config
}

// NewRuntime creates a Runtime backed by Docker.
func NewRuntime(cfg *config.Config) (Runtime, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	rt := &DockerRuntime{client: client, cfg: cfg}
	// Background image pull
	go func() {
		log.Println("Ensuring sing-box Docker image...")
		if err := client.EnsureImage(context.Background(), "ghcr.io/sagernet/sing-box:latest", ""); err != nil {
			log.Printf("Warning: failed to ensure sing-box image: %v", err)
		}
	}()
	return rt, nil
}

func (d *DockerRuntime) Start(ctx context.Context, name, configPath string) (string, error) {
	containerName := "singbox-" + name

	// Check state
	state, _ := d.client.GetContainerState(ctx, containerName)
	if state == "running" {
		return state, fmt.Errorf("container %s is already running", containerName)
	}
	if state != "" {
		_ = d.client.ContainerRemove(ctx, containerName, true)
	}

	hostConfigPath := configPath
	if resolved, err := d.cfg.ResolveHostConfigDir(configPath); err == nil {
		hostConfigPath = resolved
	}

	containerConfig := map[string]interface{}{
		"Image": "sing-box",
		"Cmd":   []string{"run", "-c", "/etc/sing-box/config.json"},
	}
	hostConfig := map[string]interface{}{
		"Binds":       []string{hostConfigPath + ":/etc/sing-box/config.json:ro"},
		"NetworkMode": "host",
		"CapAdd":      []string{"NET_ADMIN", "SYS_MODULE"},
	}

	id, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	if err := d.client.ContainerStart(ctx, id); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	return id, nil
}

func (d *DockerRuntime) Stop(ctx context.Context, name string, timeout *int) error {
	containerName := "singbox-" + name
	state, err := d.client.GetContainerState(ctx, containerName)
	if err != nil {
		return err
	}
	if state == "" {
		return nil
	}
	t := 10
	if timeout != nil {
		t = *timeout
	}
	return d.client.ContainerStop(ctx, containerName, &t)
}

func (d *DockerRuntime) Status(ctx context.Context, name string) (bool, string, error) {
	containerName := "singbox-" + name
	state, err := d.client.GetContainerState(ctx, containerName)
	if err != nil || state == "" {
		return false, "", err
	}
	return state == "running", state, nil
}

func (d *DockerRuntime) Logs(ctx context.Context, name, tail string) (string, error) {
	containerName := "singbox-" + name
	if tail == "" {
		tail = "100"
	}
	return d.client.ContainerLogs(ctx, containerName, tail)
}

func (d *DockerRuntime) Version(_ context.Context) (string, error) {
	return "sing-box 1.10.0", nil
}

func (d *DockerRuntime) List(ctx context.Context) ([]InstanceInfo, error) {
	containers, err := d.client.ListContainers(ctx, "singbox")
	if err != nil {
		return nil, err
	}
	result := make([]InstanceInfo, len(containers))
	for i, c := range containers {
		result[i] = InstanceInfo{
			Name:    c.Name,
			ID:      c.ContainerID,
			Running: c.State == "running",
			State:   c.State,
		}
	}
	return result, nil
}

func (d *DockerRuntime) Close() error {
	return d.client.Close()
}
```

- [ ] **Step 2: Build to verify compilation**

```bash
cd server && go build ./...
```

Expected: Compiles successfully (on Linux without build tag).

- [ ] **Step 3: Commit**

```bash
git add server/internal/singbox/runtime_docker.go
git commit -m "feat: implement DockerRuntime"
```

---

### Task 3: Implement NativeRuntime

**Files:**
- Create: `server/internal/singbox/runtime_native.go`

- [ ] **Step 1: Write NativeRuntime**

```go
// Package singbox provides services for managing sing-box configuration and containers.
//
//go:build openwrt

package singbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"singbox-config-service/internal/pkg/config"
)

const (
	runsDir   = "singbox/run"
	logSuffix = ".log"
	pidSuffix = ".pid"
)

// NativeRuntime manages sing-box instances as OS processes.
type NativeRuntime struct {
	binaryPath string
	dataDir    string
}

// NewRuntime creates a Runtime backed by native OS processes.
func NewRuntime(cfg *config.Config) (Runtime, error) {
	binaryPath := cfg.GetSingboxBinPath()
	if binaryPath == "" {
		var err error
		binaryPath, err = exec.LookPath("sing-box")
		if err != nil {
			return nil, fmt.Errorf("sing-box binary not found: "+
				"set --singbox-bin or install sing-box in PATH")
		}
	}
	return &NativeRuntime{
		binaryPath: binaryPath,
		dataDir:    cfg.GetDataDir(),
	}, nil
}

func (n *NativeRuntime) runDir() string {
	return filepath.Join(n.dataDir, runsDir)
}

func (n *NativeRuntime) pidFile(name string) string {
	return filepath.Join(n.runDir(), name+pidSuffix)
}

func (n *NativeRuntime) logFile(name string) string {
	return filepath.Join(n.runDir(), name+logSuffix)
}

func (n *NativeRuntime) Start(ctx context.Context, name, configPath string) (string, error) {
	if err := os.MkdirAll(n.runDir(), 0755); err != nil {
		return "", fmt.Errorf("create run dir: %w", err)
	}

	// Check if already running
	if running, id, _ := n.checkRunning(name); running {
		return id, fmt.Errorf("instance %s is already running", name)
	}

	logPath := n.logFile(name)
	logF, err := os.Create(logPath)
	if err != nil {
		return "", fmt.Errorf("create log file: %w", err)
	}
	// logF is passed to cmd; it will be closed when the process exits

	cmd := exec.CommandContext(ctx, n.binaryPath, "run", "-c", configPath)
	cmd.Stdout = logF
	cmd.Stderr = logF

	if err := cmd.Start(); err != nil {
		logF.Close()
		return "", fmt.Errorf("start sing-box: %w", err)
	}

	// Write PID file
	pid := cmd.Process.Pid
	pidStr := strconv.Itoa(pid)
	if err := os.WriteFile(n.pidFile(name), []byte(pidStr+"\n"), 0644); err != nil {
		// Process already started; warn but don't fail
		_, _ = fmt.Fprintf(logF, "warning: failed to write pid file: %v\n", err)
	}

	// Release log file — it stays open in the child process
	logF.Close()

	return fmt.Sprintf("pid:%d", pid), nil
}

func (n *NativeRuntime) Stop(ctx context.Context, name string, timeout *int) error {
	running, pidStr, err := n.checkRunning(name)
	if err != nil {
		return err
	}
	if !running {
		return nil
	}

	pid := mustParsePID(pidStr)
	proc, err := os.FindProcess(pid)
	if err != nil {
		return n.cleanupPID(name)
	}

	// SIGTERM
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return n.cleanupPID(name)
	}

	// Wait for process to exit
	t := 10
	if timeout != nil {
		t = *timeout
	}
	waitCh := make(chan bool, 1)
	go func() {
		proc.Wait()
		waitCh <- true
	}()
	select {
	case <-waitCh:
		// Graceful exit
	case <-time.After(time.Duration(t) * time.Second):
		// Force kill
		_ = proc.Kill()
	case <-ctx.Done():
		_ = proc.Kill()
	}

	return n.cleanupPID(name)
}

func (n *NativeRuntime) Status(ctx context.Context, name string) (bool, string, error) {
	return n.checkRunning(name)
}

func (n *NativeRuntime) Logs(ctx context.Context, name, tail string) (string, error) {
	logPath := n.logFile(name)
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if tail == "" {
		return string(data), nil
	}
	lines := strings.Split(string(data), "\n")
	tailN, err := strconv.Atoi(tail)
	if err != nil || tailN <= 0 || tailN >= len(lines) {
		return string(data), nil
	}
	return strings.Join(lines[len(lines)-tailN:], "\n"), nil
}

func (n *NativeRuntime) Version(_ context.Context) (string, error) {
	cmd := exec.Command(n.binaryPath, "version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown", fmt.Errorf("get version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (n *NativeRuntime) List(ctx context.Context) ([]InstanceInfo, error) {
	entries, err := os.ReadDir(n.runDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	seen := make(map[string]bool)
	var result []InstanceInfo
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), pidSuffix)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		running, id, _ := n.checkRunning(name)
		state := "stopped"
		if running {
			state = "running"
		}
		result = append(result, InstanceInfo{
			Name:    name,
			ID:      id,
			Running: running,
			State:   state,
		})
	}
	return result, nil
}

func (n *NativeRuntime) Close() error {
	return nil
}

// checkRunning checks if the process identified by name is running.
func (n *NativeRuntime) checkRunning(name string) (bool, string, error) {
	data, err := os.ReadFile(n.pidFile(name))
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false, "", n.cleanupPID(name)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, "", n.cleanupPID(name)
	}
	// Signal 0 checks existence without sending a signal
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false, "", n.cleanupPID(name)
	}
	return true, fmt.Sprintf("pid:%d", pid), nil
}

// cleanupPID removes the PID file for an instance.
func (n *NativeRuntime) cleanupPID(name string) error {
	return os.Remove(n.pidFile(name))
}

func mustParsePID(s string) int {
	pid, err := strconv.Atoi(strings.TrimPrefix(s, "pid:"))
	if err != nil {
		return -1
	}
	return pid
}


```

- [ ] **Step 2: Verify it compiles with the openwrt tag**

```bash
cd server && go build -tags openwrt ./...
```

Expected: Compiles without errors (Docker SDK not referenced).

- [ ] **Step 3: Verify default build still works**

```bash
cd server && go build ./...
```

Expected: Compiles without errors (DockerRuntime used).

- [ ] **Step 4: Commit**

```bash
git add server/internal/singbox/runtime_native.go
git commit -m "feat: implement NativeRuntime with PID file management"
```

---

### Task 4: Refactor singbox/service.go to Use Runtime

**Files:**
- Modify: `server/internal/singbox/service.go`
- Modify: `server/internal/singbox/interfaces.go`
- Keep: `server/internal/singbox/handler.go` (unchanged)
- Keep: `server/internal/singbox/register.go` (unchanged)

- [ ] **Step 1: Replace `ContainerManager` with `Runtime` in service.go**

Current imports include `"singbox-config-service/internal/pkg/docker"` — remove it.

Current struct:
```go
type Service struct {
    docker ContainerManager
    cfg    *config.Config
}
```

Change to:
```go
type Service struct {
    runtime Runtime
    cfg     *config.Config
}
```

Update constructor:
```go
func NewService(runtime Runtime, cfg *config.Config) *Service {
    return &Service{
        runtime: runtime,
        cfg:     cfg,
    }
}
```

Replace all method bodies with Runtime calls:

`RunContainer()`:
```go
func (s *Service) RunContainer() (string, error) {
    configPath := filepath.Join(s.cfg.GetSingboxDir(), "config.json")
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return "", fmt.Errorf("config file not found, please save config first")
    }
    return s.runtime.Start(context.TODO(), "default", configPath)
}
```

`StopContainer()`:
```go
func (s *Service) StopContainer() error {
    timeout := 10
    return s.runtime.Stop(context.TODO(), "default", &timeout)
}
```

`ContainerStatus()`:
```go
func (s *Service) ContainerStatus() (running bool, containerID string) {
    running, id, err := s.runtime.Status(context.TODO(), "default")
    if err != nil {
        return false, ""
    }
    return running, id
}
```

`ContainerLogs()`:
```go
func (s *Service) ContainerLogs() string {
    logs, err := s.runtime.Logs(context.TODO(), "default", "100")
    if err != nil {
        return fmt.Sprintf("Error getting logs: %v", err)
    }
    return logs
}
```

`EnsureImage()`:
```go
func (s *Service) EnsureImage() error {
    // Image pull moved to DockerRuntime.NewRuntime.
    // In DockerRuntime it happens in background; no-op here.
    return nil
}
```

`RunNamedContainer(name)`:
```go
func (s *Service) RunNamedContainer(name string) (string, error) {
    configPath := s.getNamedConfigPath(name)
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        return "", fmt.Errorf("config for '%s' not found", name)
    }
    return s.runtime.Start(context.TODO(), name, configPath)
}
```

`StopNamedContainer(name)`:
```go
func (s *Service) StopNamedContainer(name string) error {
    timeout := 10
    return s.runtime.Stop(context.TODO(), name, &timeout)
}
```

`NamedContainerStatus(name)`:
```go
func (s *Service) NamedContainerStatus(name string) (running bool, containerID string) {
    running, id, err := s.runtime.Status(context.TODO(), name)
    if err != nil {
        return false, ""
    }
    return running, id
}
```

`NamedContainerLogs(name)`:
```go
func (s *Service) NamedContainerLogs(name string) string {
    logs, err := s.runtime.Logs(context.TODO(), name, "100")
    if err != nil {
        return fmt.Sprintf("Error getting logs: %v", err)
    }
    return logs
}
```

`ListAllContainers()`:
```go
func (s *Service) ListAllContainers() ([]InstanceInfo, error) {
    return s.runtime.List(context.TODO())
}
```

Keep these methods unchanged (they deal only with config files on disk):
- `SaveConfig`
- `GetConfig`
- `SaveNamedConfig`
- `LoadNamedConfig`
- `DeleteNamedConfig`
- `ListNamedConfigs` (reads filesystem dirs, calls `NamedContainerStatus`)
- `CheckNamedConfig`
- `GetVersion`
- `getNamedConfigPath`

In `ListNamedConfigs`, change the return type from `[]docker.ContainerInfo` to `[]InstanceInfo` since we no longer import docker.

- [ ] **Step 2: Update interfaces.go**

Replace the entire contents with a single re-export if needed, or remove it since `Runtime` is now defined in `runtime.go`:

```go
package singbox

// Runtime is defined in runtime.go — this file is kept for backward compatibility.
```

Or simply keep it empty. The `ContainerManager` interface is no longer needed.

- [ ] **Step 3: Fix import of models.go if it references docker types**

Check if `models.go`, `handler.go` or `register.go` import docker. If any do, update to use `InstanceInfo` from `runtime.go` instead of `docker.ContainerInfo`.

In `models.go`, check that `ListAllContainers` return type matches what handler expects. If handler uses `docker.ContainerInfo`, change to `InstanceInfo`.

In `handler.go`, `ListAllContainers`:
```go
func (h *Handler) ListAllContainers(c *gin.Context) {
    containers, err := h.service.ListAllContainers()
    if err != nil {
        c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
            Error:   "Failed to list containers",
            Message: err.Error(),
        })
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "containers": containers,
    })
}
```

The handler doesn't import docker types — it just returns JSON. No change needed.

- [ ] **Step 4: Build and test**

```bash
cd server && go build ./... && go test ./...
```

Expected: All compiles and tests pass.

- [ ] **Step 5: Commit**

```bash
git add server/internal/singbox/service.go server/internal/singbox/interfaces.go
git commit -m "refactor: singbox service uses Runtime interface"
```

---

### Task 5: TempRuntime Interface + Implementations for Speedtest

**Files:**
- Create: `server/internal/speedtest/runtimer.go`
- Create: `server/internal/speedtest/runtimer_docker.go`
- Create: `server/internal/speedtest/runtimer_native.go`

- [ ] **Step 1: Write TempRuntime interface**

```go
// Package speedtest provides a speed testing service for proxy nodes.
package speedtest

import (
	"context"
	"time"
)

// TempRuntime manages short-lived sing-box instances for speed testing.
type TempRuntime interface {
	// StartTemp starts a temporary sing-box instance with the given config.
	StartTemp(ctx context.Context, configPath string) (id string, err error)

	// StopTemp stops and cleans up a temporary instance.
	StopTemp(ctx context.Context, id string) error

	// WaitTempReady blocks until the instance accepts TCP connections on the given port.
	WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error

	// GetTempLogs returns collected log output from a temporary instance.
	GetTempLogs(ctx context.Context, id string) string
}
```

- [ ] **Step 2: Write DockerTempRuntime**

```go
// Package speedtest provides a speed testing service for proxy nodes.
//
//go:build !openwrt

package speedtest

import (
	"context"
	"fmt"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
)

const speedTestContainerName = "sing-box-speedtest"
const singBoxImage = "ghcr.io/sagernet/sing-box:v1.13.5"

// DockerTempRuntime creates temporary sing-box containers for speed tests.
type DockerTempRuntime struct {
	docker docker.ContainerAPI
	cfg    *config.Config
}

// NewTempRuntime creates a TempRuntime backed by Docker containers.
func NewTempRuntime(cfg *config.Config) TempRuntime {
	client, err := docker.NewClient()
	if err != nil {
		// Speedtest won't work, but operations continue; each method checks for nil.
		return &DockerTempRuntime{cfg: cfg}
	}
	return &DockerTempRuntime{docker: client, cfg: cfg}
}

func (d *DockerTempRuntime) StartTemp(ctx context.Context, configPath string) (string, error) {
	if d.docker == nil {
		return "", fmt.Errorf("Docker client not available")
	}
	hostConfigPath := configPath
	if resolved, err := d.cfg.ResolveHostConfigDir(configPath); err == nil {
		hostConfigPath = resolved
	}

	containerConfig := map[string]interface{}{
		"Image": singBoxImage,
		"Cmd":   []string{"-D", "/var/lib/sing-box", "-C", "/etc/sing-box/", "run"},
	}
	hostConfig := map[string]interface{}{
		"Binds":       []string{hostConfigPath + ":/etc/sing-box/config.json:ro"},
		"NetworkMode": "host",
		"CapAdd":      []string{"NET_ADMIN"},
	}

	d.cleanupContainer(ctx)

	for attempt := 0; attempt < 5; attempt++ {
		id, err := d.docker.ContainerCreate(ctx, containerConfig, hostConfig, speedTestContainerName)
		if err == nil {
			if err := d.docker.ContainerStart(ctx, id); err != nil {
				continue
			}
			return id, nil
		}
		if !isConflictError(err) {
			d.cleanupContainer(ctx)
			continue
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", fmt.Errorf("failed to create speedtest container after retries")
}

func (d *DockerTempRuntime) StopTemp(ctx context.Context, id string) error {
	d.cleanupContainer(ctx)
	return nil
}

func (d *DockerTempRuntime) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	return waitProxyReady(ctx, port, timeout)
}

func (d *DockerTempRuntime) GetTempLogs(ctx context.Context, id string) string {
	if d.docker == nil {
		return "Docker client not available"
	}
	logs, err := d.docker.ContainerLogs(ctx, speedTestContainerName, "50")
	if err != nil {
		return fmt.Sprintf("Error getting logs: %v", err)
	}
	return logs
}

func (d *DockerTempRuntime) cleanupContainer(ctx context.Context) {
	if d.docker == nil {
		return
	}
	_ = d.docker.ContainerRemove(ctx, speedTestContainerName, true)
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return contains(s, "already in use") || contains(s, "Conflict")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

Note: The `waitProxyReady` function already exists in `service.go`. We'll extract it later.

- [ ] **Step 3: Write NativeTempRuntime**

```go
// Package speedtest provides a speed testing service for proxy nodes.
//
//go:build openwrt

package speedtest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// NativeTempRuntime creates temporary sing-box processes for speed tests.
type NativeTempRuntime struct {
	binaryPath string
}

// NewTempRuntime creates a TempRuntime backed by native OS processes.
func NewTempRuntime(cfg *config.Config) TempRuntime {
	binaryPath := cfg.GetSingboxBinPath()
	if binaryPath == "" {
		if p, err := exec.LookPath("sing-box"); err == nil {
			binaryPath = p
		}
	}
	return &NativeTempRuntime{binaryPath: binaryPath}
}

type tempInstance struct {
	cmd    *exec.Cmd
	logBuf *bytes.Buffer
}

var instances = make(map[string]*tempInstance)

func (n *NativeTempRuntime) StartTemp(ctx context.Context, configPath string) (string, error) {
	if n.binaryPath == "" {
		return "", fmt.Errorf("sing-box binary not configured")
	}

	var logBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, n.binaryPath, "run", "-c", configPath)
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start: %w", err)
	}

	pid := cmd.Process.Pid
	id := fmt.Sprintf("pid:%d", pid)
	instances[id] = &tempInstance{cmd: cmd, logBuf: &logBuf}
	return id, nil
}

func (n *NativeTempRuntime) StopTemp(ctx context.Context, id string) error {
	inst, ok := instances[id]
	if !ok {
		return nil
	}
	pid := mustParsePid(id)
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		_ = proc.Kill()
		inst.cmd.Wait()
	}
	delete(instances, id)
	return nil
}

func (n *NativeTempRuntime) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	// Delegate to the shared waitProxyReady function in service.go
	return waitProxyReady(ctx, port, timeout)
}

func (n *NativeTempRuntime) GetTempLogs(ctx context.Context, id string) string {
	inst, ok := instances[id]
	if !ok {
		return "instance not found"
	}
	return inst.logBuf.String()
}

func mustParsePid(id string) int {
	s := strings.TrimPrefix(id, "pid:")
	pid, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return pid
}
```

- [ ] **Step 4: Build both variants**

```bash
cd server && go build ./... && go build -tags openwrt ./...
```

Both should compile without errors.

- [ ] **Step 5: Commit**

```bash
git add server/internal/speedtest/runtimer.go server/internal/speedtest/runtimer_docker.go server/internal/speedtest/runtimer_native.go
git commit -m "feat: add TempRuntime interface with Docker and Native implementations"
```

---

### Task 6: Refactor speedtest/service.go to Use TempRuntime

**Files:**
- Modify: `server/internal/speedtest/service.go`
- Modify: `server/internal/speedtest/interfaces.go`

- [ ] **Step 1: Update interfaces.go**

`TempRuntime` is already defined in `runtimer.go`. Remove the old `ContainerManager` interface
from `interfaces.go` and keep only the other interfaces:

```go
package speedtest

import "singbox-config-service/internal/pkg/types"

// NodeProvider provides proxy nodes to test.
type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}

// SpeedTestResultSaver persists speed test results.
type SpeedTestResultSaver interface {
	SaveSpeedTestResults(results []types.SpeedTestUpdate) error
}
```

- [ ] **Step 2: Update service.go**

Change struct:
```go
type Service struct {
	tempRuntime  TempRuntime
	cfg          *config.Config
	nodeProvider NodeProvider
	resultSaver  SpeedTestResultSaver
	state        *SpeedTestState
	mu           sync.Mutex
	cancel       context.CancelFunc
}
```

Remove import of `"singbox-config-service/internal/pkg/docker"` (if present) — not needed anymore.

Change constructor:
```go
func NewService(tempRuntime TempRuntime, cfg *config.Config) *Service {
	return &Service{
		tempRuntime: tempRuntime,
		cfg:         cfg,
		state:       &SpeedTestState{},
	}
}
```

In `testOneNode`, replace Docker-specific code with `tempRuntime` calls:

Replace:
```go
id, err = s.docker.ContainerCreate(ctx, containerConfig, hostConfig, speedTestContainerName)
...
if err := s.docker.ContainerStart(ctx, id); err != nil {
```

With:
```go
// Remove hostConfigPath resolution and containerConfig/hostConfig construction
// Just pass configPath directly
id, err := s.tempRuntime.StartTemp(ctx, cfgPath)
if err != nil {
    return 0, 0, "", fmt.Errorf("start temp instance: %w", err)
}
```

Replace `waitProxyReady(ctx, port, 10*time.Second)` — this is already abstracted. Keep it as is; it calls the shared function.

Replace cleanup:
```go
s.cleanupContainer()
```
With:
```go
s.tempRuntime.StopTemp(ctx, id)
```

The `cleanupContainer()` and `getContainerLogs()` methods can be removed or updated.

Update `getContainerLogs()` if still needed:
```go
func (s *Service) getContainerLogs(id string) string {
    return s.tempRuntime.GetTempLogs(context.Background(), id)
}
```

Remove the `cleanupContainer()` method entirely — `StopTemp` handles it.

Remove the containerConfig/hostConfig building code from `testOneNode` since `TempRuntime` handles that internally.

- [ ] **Step 3: Ensure `waitProxyReady` is accessible**

`waitProxyReady` is a package-level function in `service.go`. Keep it there — both Docker and Native temp runtimes can call it since it only does TCP port polling (no Docker dependency).

- [ ] **Step 4: Build and test**

```bash
cd server && go build ./... && go test ./...
```

Expected: All compiles and tests pass.

- [ ] **Step 5: Commit**

```bash
git add server/internal/speedtest/service.go server/internal/speedtest/interfaces.go
git commit -m "refactor: speedtest service uses TempRuntime interface"
```

---

### Task 7: Update main.go

**Files:**
- Modify: `server/main.go`

- [ ] **Step 1: Replace Docker client creation with Runtime factory**

Remove these imports (if present):
```go
"singbox-config-service/internal/pkg/docker"
```

Remove:
```go
dockerClient, err := docker.NewClient()
if err != nil {
    log.Printf("Warning: Failed to create Docker client: %v", err)
    log.Println("Docker-dependent features will not be available")
}
if dockerClient != nil {
    defer dockerClient.Close()
}
```

Replace with:
```go
// Create sing-box runtime (implementation selected by build tag)
singboxRT, err := singbox.NewRuntime(cfg)
if err != nil {
    log.Printf("Warning: Failed to create sing-box runtime: %v", err)
    log.Println("sing-box management features will not be available")
}
if singboxRT != nil {
    defer singboxRT.Close()
}

// Create sing-box service
singboxSvc := singbox.NewService(singboxRT, cfg)

// Create speedtest runtime + service
speedtestRT := speedtest.NewTempRuntime(cfg)
speedtestSvc := speedtest.NewService(speedtestRT, cfg)
```

Remove background image pull:
```go
if dockerClient != nil {
    go func() {
        log.Println("Pulling sing-box image in background...")
        ...
    }()
}
```
This is now handled inside `DockerRuntime.NewRuntime`.

- [ ] **Step 2: Wire up the --singbox-bin flag**

At the top of `main()`:
```go
serveDashboard := flag.Bool("dashboard", false, "Serve embedded frontend dashboard")
singboxBin := flag.String("singbox-bin", "", "Path to sing-box binary (native mode)")
flag.Parse()
```

Then after `cfg, err := config.Init()`:
```go
cfg.SetSingboxBinPath(*singboxBin)
```

- [ ] **Step 3: Build and test**

```bash
cd server && go build ./... && go test ./...
```

Expected: Compiles and all tests pass.

- [ ] **Step 4: Commit**

```bash
git add server/main.go
git commit -m "refactor: main.go uses Runtime factories instead of direct Docker client"
```

---

### Task 8: Build System and Makefile

**Files:**
- Modify: `Makefile` (or create if not existing)

- [ ] **Step 1: Add Makefile with build targets**

```makefile
.PHONY: build build-linux build-openwrt-mips test clean

# Default: Linux with Docker support
build: build-linux

build-linux:
	go mod tidy
	go build -ldflags="-s -w" -o bin/singbox-ui-linux .

# OpenWrt (mipsle softfloat) — native mode, no Docker
build-openwrt-mips:
	GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
	go build -tags openwrt -ldflags="-s -w" -o bin/singbox-ui-mipsle .

# OpenWrt (arm64)
build-openwrt-arm64:
	GOOS=linux GOARCH=arm64 \
	go build -tags openwrt -ldflags="-s -w" -o bin/singbox-ui-arm64 .

test:
	go test ./... -v

clean:
	rm -rf bin/
```

- [ ] **Step 2: Verify builds**

```bash
cd server && go build ./... && go test ./...
```

Expected: Compiles and all tests pass.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add Makefile with OpenWrt cross-compile targets"
```

---

### Task 9: Verify NativeRuntime Works (Manual Test on Linux)

**Files:** None (inspection only)

- [ ] **Step 1: Build native variant on Linux**

```bash
cd server && go build -tags openwrt -o singbox-ui-native .
```

- [ ] **Step 2: Test with sing-box binary**

If sing-box is installed on the dev machine:
```bash
./singbox-ui-native --singbox-bin $(which sing-box)
```

Expected: Server starts, no Docker-related errors.

- [ ] **Step 3: Test native mode without binary**

```bash
./singbox-ui-native
```

Expected: Log message: "sing-box binary not found"

- [ ] **Step 4: Document limitations**

Add a note in the code or README that `--singbox-bin` is mandatory when running on OpenWrt (or the binary must be in `$PATH`).
