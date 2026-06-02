# Native Runtime: Non-Docker Sing-Box & OpenWrt Support

**Date:** 2026-06-02
**Status:** Specified

## Problem

The sing-box UI currently requires Docker to run sing-box. All sing-box lifecycle
operations (start, stop, status, logs, speedtest) go through the Docker SDK. Two
deployment scenarios are not supported:

1. **OpenWrt** — Docker is not available; sing-box runs as a native process.
2. **Linux without Docker** — users who prefer direct process management over containers.

The backend (Go server) must also be able to run directly on OpenWrt (cross-compiled
for MIPS/ARM architectures).

## Scope

This spec covers changes to the Go backend only. The frontend API contract is unchanged;
all existing HTTP endpoints remain the same. The Docker SDK dependency is conditionally
excluded via Go build tags for the OpenWrt build.

---

## 1. Approach: Build-Tag-Selected Runtime

Two implementations of a `Runtime` interface are selected at compile time via Go
build tags:

| Build tag | Runtime | Platform | `go build` |
|-----------|---------|----------|------------|
| (default) | `DockerRuntime` | Linux (amd64/arm64) | `go build` |
| `openwrt` | `NativeRuntime` | OpenWrt (mips/arm) | `go build -tags openwrt` |

### Dependency strategy

Only files included in the current build are compiled. Since `runtime_docker.go`
has `//go:build !openwrt`, the Docker SDK is never compiled into the OpenWrt binary.
`go mod tidy` must be run _without_ the `openwrt` tag to keep Docker SDK in `go.mod`;
it is harmless dead weight for OpenWrt builds.

---

## 2. Runtime Interface

**File:** `server/internal/singbox/runtime.go` (no build tag — always compiled)

```go
// Runtime abstracts sing-box process lifecycle for Docker and native modes.
type Runtime interface {
    Start(ctx context.Context, name string, configPath string) (id string, err error)
    Stop(ctx context.Context, name string, timeout *int) error
    Status(ctx context.Context, name string) (running bool, id string, err error)
    Logs(ctx context.Context, name string, tail string) (string, error)
    Version(ctx context.Context) (string, error)
    List(ctx context.Context) ([]InstanceInfo, error)
    Close() error
}

type InstanceInfo struct {
    Name    string `json:"name"`
    ID      string `json:"container_id,omitempty"`
    Running bool   `json:"running"`
    State   string `json:"state,omitempty"`
}
```

`name` is the logical identifier (e.g. `"default"`, `"my-instance"`):
- `DockerRuntime` maps it to container name `singbox-<name>`.
- `NativeRuntime` maps it to PID file `DATA_DIR/singbox/run/<name>.pid`.

---

## 3. DockerRuntime

**File:** `server/internal/singbox/runtime_docker.go` (`//go:build !openwrt`)

Wraps the existing `internal/pkg/docker` package. All Docker-specific logic currently
in `singbox/service.go` moves here:

| Method | Source (current `service.go`) |
|--------|-------------------------------|
| `Start` | `RunContainer` / `RunNamedContainer` |
| `Stop` | `StopContainer` / `StopNamedContainer` |
| `Status` | `ContainerStatus` / `NamedContainerStatus` |
| `Logs` | `ContainerLogs` / `NamedContainerLogs` |
| `Version` | `GetVersion` (hardcoded) or `docker exec` |
| `List` | `ListAllContainers` + `ListNamedConfigs` |

### Factory

```go
func NewRuntime(cfg *config.Config) (Runtime, error) {
    client, err := docker.NewClient()
    if err != nil {
        return nil, fmt.Errorf("docker client: %w", err)
    }
    rt := &DockerRuntime{client: client, cfg: cfg}
    // Background image pull — moved from main.go
    go rt.ensureImage(context.Background())
    return rt, nil
}
```

---

## 4. NativeRuntime

**File:** `server/internal/singbox/runtime_native.go` (`//go:build openwrt`)

Manages sing-box as an OS process via `os/exec`.

### Process lifecycle

| Operation | Implementation |
|-----------|----------------|
| **Start** | `exec.Command(binaryPath, "run", "-c", configPath)` → `cmd.Start()` → write PID to `DATA_DIR/singbox/run/<name>.pid` |
| **Stop** | Read PID file → `process.Signal(syscall.SIGTERM)` → wait up to `timeout` seconds → `SIGKILL` if still alive |
| **Status** | Read PID file → `process.Signal(syscall.Signal(0))` (checks process existence) |
| **Logs** | Read `DATA_DIR/singbox/run/<name>.log` (captured stdout/stderr), tail N lines |
| **Version** | `exec.Command(binaryPath, "version").Output()` |
| **List** | Scan `DATA_DIR/singbox/run/*.pid`, check each PID |

### Log capture

stdout and stderr of the sing-box process are redirected to
`DATA_DIR/singbox/run/<name>.log`. This is independent of sing-box's own
`log.output` config option — it captures all process output reliably.

A simple size guard (e.g. truncate at 1 MB) prevents unbounded log growth.

### PID file layout

```
DATA_DIR/singbox/run/
├── default.pid       # contains "1234\n"
├── my-instance.pid
└── my-instance.log
```

### Binary path resolution

1. `--singbox-bin` CLI flag value (if set).
2. `exec.LookPath("sing-box")` — search `$PATH`.
3. Error if not found.

### Factory

```go
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
```

---

## 5. Speedtest TempRuntime

Speed testing creates short-lived sing-box instances. A second, simpler interface
covers this use case.

**File:** `server/internal/speedtest/runtimer.go` (no build tag)

```go
type TempRuntime interface {
    StartTemp(ctx context.Context, configPath string) (id string, err error)
    StopTemp(ctx context.Context, id string) error
    WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error
    GetTempLogs(ctx context.Context, id string) string
}
```

### DockerTempRuntime (`//go:build !openwrt`)

Wraps existing Docker container create/start/remove logic from `speedtest/service.go`.

### NativeTempRuntime (`//go:build openwrt`)

- `StartTemp`: `exec.CommandContext` with context for timeout.
- `StopTemp`: kill process by PID.
- `WaitTempReady`: TCP port polling (unchanged logic, already abstracted as `waitProxyReady` in `service.go`).
- `GetTempLogs`: return captured stdout/stderr buffer.

### Factory

```go
// runtimer_docker.go — //go:build !openwrt
func NewTempRuntime(cfg *config.Config) TempRuntime {
    return &DockerTempRuntime{/* ... */}
}

// runtimer_native.go — //go:build openwrt
func NewTempRuntime(cfg *config.Config) TempRuntime {
    return &NativeTempRuntime{binaryPath: cfg.GetSingboxBinPath()}
}
```

---

## 6. Changes to Existing Files

### `server/internal/singbox/service.go`

- `ContainerManager` field → `Runtime` field.
- Constructor: `NewService(runtime Runtime, cfg *config.Config)`.
- All container methods (`RunContainer`, `StopContainer`, `ContainerStatus`, etc.)
  become thin wrappers around `s.runtime.Start()`, `s.runtime.Stop()`, etc.
- `SaveConfig`, `GetConfig`, `SaveNamedConfig`, `LoadNamedConfig`, `DeleteNamedConfig`,
  `CheckNamedConfig` — unchanged (they only deal with config files on disk).

### `server/internal/singbox/interfaces.go`

Removed. The `ContainerManager` interface is superseded by `Runtime` in `runtime.go`.

### `server/internal/singbox/handler.go`

No changes. Handlers call `service.RunContainer()` etc. — they do not know about the
underlying Runtime implementation.

### `server/internal/singbox/register.go`

No changes. Routes remain identical.

### `server/internal/speedtest/service.go`

- `ContainerManager` field → `TempRuntime` field.
- All Docker-specific code (container create, start, remove, wait) → `tempRuntime` calls.
- `buildSpeedTestConfig`, `pickFreePort`, `waitProxyReady`, `newProxyClient` — unchanged.

### `server/internal/speedtest/interfaces.go`

Updated. `ContainerManager` → `TempRuntime`.

### `server/internal/pkg/config/config.go`

```go
type Config struct {
    // existing fields...
    singboxBinPath string
}

func (c *Config) SetSingboxBinPath(p string) { c.singboxBinPath = p }
func (c *Config) GetSingboxBinPath() string  { return c.singboxBinPath }
```

### `server/main.go`

- Remove `import "singbox-config-service/internal/pkg/docker"`.
- Remove `dockerClient` creation and `defer dockerClient.Close()`.
- Remove background image pull goroutine (moves to `DockerRuntime`).
- Add `--singbox-bin` flag parsing.
- Replace `singbox.NewService(dockerClient, cfg)` with:
  ```go
  rt := singbox.NewRuntime(cfg)
  defer rt.Close()
  singboxSvc := singbox.NewService(rt, cfg)
  ```
- Replace `speedtest.NewService(dockerClient, cfg)` with:
  ```go
  tempRT := speedtest.NewTempRuntime(cfg)
  speedtestSvc := speedtest.NewService(tempRT, cfg)
  ```

### `server/internal/pkg/docker/client.go`

Unchanged. Still used by `DockerRuntime`.

---

## 7. New Files Summary

| File | Build tag | Purpose |
|------|-----------|---------|
| `server/internal/singbox/runtime.go` | (none) | `Runtime` interface + `InstanceInfo` |
| `server/internal/singbox/runtime_docker.go` | `!openwrt` | `DockerRuntime` + `NewRuntime` factory |
| `server/internal/singbox/runtime_native.go` | `openwrt` | `NativeRuntime` + `NewRuntime` factory |
| `server/internal/speedtest/runtimer.go` | (none) | `TempRuntime` interface |
| `server/internal/speedtest/runtimer_docker.go` | `!openwrt` | `DockerTempRuntime` + `NewTempRuntime` factory |
| `server/internal/speedtest/runtimer_native.go` | `openwrt` | `NativeTempRuntime` + `NewTempRuntime` factory |

---

## 8. CLI Flag

| Flag | Default | Description |
|------|---------|-------------|
| `--singbox-bin` | `""` | Path to the sing-box binary (required in native mode; auto-searches `$PATH` if empty) |

---

## 9. Build & Deploy

### Linux (default, Docker mode)

```bash
go build -ldflags="-s -w" -o singbox-ui-linux .
```

### OpenWrt (native mode, cross-compile)

```bash
GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
  go build -tags openwrt -ldflags="-s -w" -o singbox-ui-mipsle .
```

Both targets use the same source tree. The Docker SDK is listed in `go.mod` but
never compiled for the OpenWrt target.

### Makefile additions (optional)

```makefile
.PHONY: build-linux build-openwrt-mips

build-linux:
    go mod tidy && go build -ldflags="-s -w" -o bin/singbox-ui-linux .

build-openwrt-mips:
    GOOS=linux GOARCH=mipsle GOMIPS=softfloat \
    go build -tags openwrt -ldflags="-s -w" -o bin/singbox-ui-mipsle .
```

---

## 10. No Changes

- **Frontend** — API contract unchanged. All existing endpoints, response shapes,
  and TypeScript types remain valid. `container_id` in JSON responses may contain
  a PID string in native mode; this is backward-compatible.
- **API routes** — unchanged.
- **Handler methods** — unchanged.
- **Docker Compose / Dockerfile** — unchanged (Docker mode still used on Linux).
- **Config tests** — existing tests unchanged. New tests for NativeRuntime.

---

## 11. Verification

1. **Linux (Docker mode, default):** `go build && ./sing-box-ui` — existing behavior unchanged.
2. **Linux (native mode, dev test):** `go build -tags openwrt && ./sing-box-ui --singbox-bin $(which sing-box)` — should start sing-box as a process.
3. **OpenWrt cross-compile:** `GOOS=linux GOARCH=mipsle go build -tags openwrt ...` — compiles successfully, no Docker references in binary.
4. **Speedtest on native:** `go build -tags openwrt` and run speedtest — starts temporary sing-box process, measures latency, cleans up.
5. **`go mod tidy` safety:** `go mod tidy` (without tags) does not remove Docker SDK from `go.mod`.
