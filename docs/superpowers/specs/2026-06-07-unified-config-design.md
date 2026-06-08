# Unified Application Configuration Design

**Date:** 2026-06-07
**Status:** Specified

## Problem

The singbox-ui backend currently scatters configuration parameters across multiple locations:

1. **`internal/pkg/config/config.go`** — `Config` struct with 5 fields, populated from env vars (`DATA_DIR`, `LISTEN_ADDR`, `HOST_DATA_DIR`) plus a CLI flag for `singbox-bin`.
2. **`internal/prober/models.go`** — `ProberConfig` with hardcoded defaults (`ProbeInterval=30`, `ProbeTimeout=5000`).
3. **`internal/speedtest/service.go`** — 3 package-level constants (`speedTestLatencyURL`, `speedTestDownloadURL`, `speedTestDuration`).
4. **`internal/scheduler/service.go`** — hardcoded `interval: 60 * time.Second`.
5. **`internal/subscription/service.go`** — `os.Getenv("SUBSCRIPTION_INSECURE_TLS")` called directly.
6. **`main.go`** — CLI flags `--dashboard` and `--singbox-bin`.

This fragmentation creates several problems:
- New parameters require adding env vars or flags, each with its own loading pattern.
- No single source of truth for configuration.
- Hard to see all configurable knobs at a glance.
- Hardcoded constants in domain packages cannot be changed by operators without code modification.

## Scope

Consolidate **all** application configuration parameters into a single YAML file with a well-defined schema. The existing `Config` struct in `internal/pkg/config` becomes the top-level `AppConfig` aggregating per-module config structs. Each domain package defines its own config struct; `AppConfig` composes them.

**In scope:**
- Server parameters (listen address, data dirs, dashboard flag, sing-box binary path).
- Prober parameters (interval, timeout, concurrency, retries, bind options).
- Speedtest parameters (latency URL, download URL, duration).
- Scheduler parameters (interval).
- Subscription parameters (insecure TLS toggle).
- YAML file loading with sensible defaults.
- Bootstrap logic to locate the config file via `DATA_DIR` env var or `--config` CLI flag.
- Backwards compatibility: if no config file exists, all defaults apply (behaviour identical to today).

**Out of scope:**
- Frontend configuration (Next.js, Tailwind, i18n messages remain as-is).
- Runtime re-loading of config (requires a restart to pick up changes).
- Environment-variable-based configuration (env is only used for bootstrap `DATA_DIR`).

## Design

### Bootstrap and Config File Location

The config file is named `config.yaml` and lives in the data directory.

**Resolution order:**
1. If `--config /path/to/config.yaml` CLI flag is provided, use that exact path.
2. Otherwise, determine `DATA_DIR`:
   - `DATA_DIR` env var if set.
   - Fall back to `os.Getwd()` (current behaviour).
3. Try `DATA_DIR/config.yaml`. If it does not exist, use defaults (compatible with existing deployments that have no config file).

**What remains in env/CLI:**
- `DATA_DIR` env var — bootstrap only (can be overridden inside config.yaml's `server.data_dir`).
- `--config` CLI flag — optional override for advanced setups.
- `TZ` env var — system/Docker concern, not application config.

### YAML Schema

```yaml
server:
  listen_addr: "0.0.0.0:7000"
  data_dir: "/home/data"
  host_data_dir: ""
  singbox_bin_path: ""
  serve_dashboard: false

prober:
  interval: 30
  timeout: 5000
  concurrent: 5
  max_results: 100
  max_retries: 2
  bind_address: ""
  bind_interface: ""

speedtest:
  latency_url: "http://www.gstatic.com/generate_204"
  download_url: "https://speed.cloudflare.com/__down?bytes=10000000"
  duration: 10                  # seconds

scheduler:
  interval: 60                  # seconds

subscription:
  insecure_tls: false
```

### Go Type Definitions

#### `internal/pkg/config/config.go` — AppConfig

```go
package config

type ServerConfig struct {
    ListenAddr      string `yaml:"listen_addr"`
    DataDir         string `yaml:"data_dir"`
    HostDataDir     string `yaml:"host_data_dir"`
    SingboxBinPath  string `yaml:"singbox_bin_path"`
    ServeDashboard  bool   `yaml:"serve_dashboard"`
}

type AppConfig struct {
    Server       ServerConfig          `yaml:"server"`
    Prober       prober.Config         `yaml:"prober"`
    Speedtest    speedtest.Config      `yaml:"speedtest"`
    Scheduler    scheduler.Config      `yaml:"scheduler"`
    Subscription subscription.Config   `yaml:"subscription"`
}
```

**Methods on AppConfig:**
- `Load(path string) (*AppConfig, error)` — reads YAML from file, merges defaults for zero-value fields.
- `Init() (*AppConfig, error)` — bootstrap: resolves DATA_DIR, locates config.yaml, calls Load, falls back to defaults if no file exists.
- `GetDataDir() string` — returns resolved data directory.
- `GetListenAddr() string` — returns server listen address.
- `GetSingboxDir() string` — returns `filepath.Join(dataDir, "singbox")`.
- `ResolveHostConfigDir(containerPath string) (string, error)` — existing method, preserved.

The `Init` signature changes from `Init() (*Config, error)` to `Init(configPath string) (*AppConfig, error)`, where `configPath` comes from the optional `--config` CLI flag.

#### `internal/prober/models.go` — Prober Config (rename ProberConfig → Config)

```go
type Config struct {
    Interval       int    `yaml:"interval"`
    Timeout        int    `yaml:"timeout"`
    Concurrent     int    `yaml:"concurrent"`
    MaxResults     int    `yaml:"max_results"`
    MaxRetries     int    `yaml:"max_retries"`
    BindAddress    string `yaml:"bind_address"`
    BindInterface  string `yaml:"bind_interface"`
}
```

The `DefaultProberConfig()` function adapts to return a `Config` with the same literal defaults as today.

#### `internal/speedtest/models.go` — New

```go
type Config struct {
    LatencyURL   string `yaml:"latency_url"`
    DownloadURL  string `yaml:"download_url"`
    Duration     int    `yaml:"duration"` // seconds
}
```

#### `internal/scheduler/models.go` — New

```go
type Config struct {
    Interval int `yaml:"interval"` // seconds
}
```

#### `internal/subscription/models.go` — Add

```go
type Config struct {
    InsecureTLS bool `yaml:"insecure_tls"`
}
```

### Default Values

Defaults are defined once, in `DefaultProberConfig()` and equivalent functions for each domain, and also in a `defaultAppConfig()` helper. These must match. The YAML loading code applies defaults for any field that is zero-valued after parsing (i.e., not set in the file). This ensures that an absent config file produces exactly the same behaviour as today.

| Section  | Field         | Default                        |
|----------|---------------|--------------------------------|
| server   | listen_addr   | `127.0.0.1:7000`               |
| prober   | interval      | `30` (seconds)                 |
| prober   | timeout       | `5000` (ms)                    |
| prober   | concurrent    | `5`                            |
| prober   | max_results   | `100`                          |
| prober   | max_retries   | `2`                            |
| speedtest| latency_url   | `http://www.gstatic.com/generate_204` |
| speedtest| download_url  | `https://speed.cloudflare.com/__down?bytes=10000000` |
| speedtest| duration      | `10` (seconds)                 |
| scheduler| interval      | `60` (seconds)                 |
| subscript| insecure_tls  | `false`                        |

### Constructor Signature Changes

```go
// prober
func NewService(cfg Config, baseDir string, resultSaver ProbeResultSaver) *Service

// speedtest
func NewService(tempRuntime TempRuntime, cfg Config) *Service

// scheduler
func New(cfg Config, subUpdater SubscriptionUpdater, containerMgr ContainerManager) *Scheduler

// subscription
func NewService(store *FileStore, cfg Config) *Service
```

No changes to handler constructors — they receive the service (which already embeds its config).

### Backwards Compatibility

1. **No config file exists:** All fields get defaults. Behaviour is identical to today's code.
2. **Partial config file:** Unset fields get defaults. An operator can create a config.yaml with only the parameters they want to change.
3. **Old env vars:** `DATA_DIR`, `LISTEN_ADDR`, `HOST_DATA_DIR` continue to be read in `Init()` **only if** the corresponding YAML field is not set. This eases migration: operators can create a config.yaml piece by piece.
4. **CLI flags:** `--dashboard` and `--singbox-bin` are removed. Their equivalents live in `server.serve_dashboard` and `server.singbox_bin_path`. A future PR could read them from CLI for convenience, but the initial implementation uses only the YAML file.

## Implementation Plan

The work proceeds in natural bottom-up order: config structs first, then the loader, then wiring in services, and finally main.go.

### Step 1: Define per-module config structs

- `internal/prober/models.go`: Rename `ProberConfig` → `Config`, add `MaxRetries`, `BindAddress`, `BindInterface`.
- `internal/speedtest/models.go`: New file with `Config` struct.
- `internal/scheduler/models.go`: New file with `Config` struct.
- `internal/subscription/models.go`: Add `Config` struct.

### Step 2: Refactor `internal/pkg/config` package

- Rename existing `Config` → `ServerConfig`.
- Create `AppConfig` struct composing all module configs.
- Add YAML loading (`AppConfig.Load(path)`).
- Add `defaultAppConfig()` returning full defaults.
- Adapt `Init()` to accept optional config path, locate file, call Load.
- Preserve existing getter methods on `AppConfig`.

### Step 3: Update service constructors

- `prober.NewService` — accept `prober.Config` instead of hardcoded defaults.
- `speedtest.NewService` — accept `speedtest.Config`, remove package-level constants.
- `scheduler.New` — accept `scheduler.Config`, remove hardcoded interval.
- `subscription.NewService` — accept `subscription.Config`, remove direct `os.Getenv`.

### Step 4: Update `main.go`

- Replace `--dashboard`, `--singbox-bin` flags with `--config`.
- Call `config.Init(*configPath)`.
- Pass relevant sub-configs to each service constructor.
- Remove `cfg.SetSingboxBinPath(*singboxBin)` — value comes from config now.

### Step 5: Update tests

- `config_test.go`: Add tests for YAML loading, default merging, partial config.
- `prober/engine_test.go`: Update `NewProber(DefaultProberConfig())` calls if signature changed.
- `speedtest/service_test.go`: Update constructors to pass `speedtest.Config`.
- `scheduler/service_test.go`: Update constructors to pass `scheduler.Config`.
- `subscription/service_test.go`: Update constructors to pass `subscription.Config`.

### Step 6: Update docker-compose.yml

- Remove `LISTEN_ADDR` from environment (goes into config.yaml).
- Keep `DATA_DIR` (needed for bootstrap) and `TZ`.
- Optionally document `--config` override in comments.

### Step 7: Generate example config.yaml

- Create `config.example.yaml` at project root with all options documented.

### Step 8: Verify — build, lint, test

```bash
cd server && go build ./... && golangci-lint run ./... && go test ./...
```

## Testing Strategy

| Layer | Tests |
|-------|-------|
| **Unit: config loading** | `TestLoadConfig_full`, `TestLoadConfig_partial`, `TestLoadConfig_fileNotFound` (defaults), `TestInit_bootstrap` |
| **Unit: each module's config defaults** | Verify `DefaultConfig()` returns expected values |
| **Integration: service wiring** | Construct each service with its sub-config, verify parameters are picked up (e.g. prober interval matches config) |
| **Regression** | Full test suite passes with zero config file present (all defaults) |

## Files Changed

| File | Change |
|------|--------|
| `server/internal/pkg/config/config.go` | Refactor: Config → ServerConfig + AppConfig + Load/Init |
| `server/internal/pkg/config/config_test.go` | New tests for YAML loading, defaults, bootstrap |
| `server/internal/prober/models.go` | Rename ProberConfig → Config, add new fields |
| `server/internal/prober/engine.go` | Use Config instead of ProberConfig |
| `server/internal/prober/service.go` | Accept Config in constructor |
| `server/internal/speedtest/models.go` | New file: Config struct |
| `server/internal/speedtest/service.go` | Accept Config, remove package constants |
| `server/internal/speedtest/interfaces.go` | No change |
| `server/internal/scheduler/models.go` | New file: Config struct |
| `server/internal/scheduler/service.go` | Accept Config in constructor |
| `server/internal/subscription/models.go` | Add Config struct |
| `server/internal/subscription/service.go` | Accept Config, replace os.Getenv |
| `server/main.go` | Replace CLI flags, wire new constructors |
| `docker-compose.yml` | Remove LISTEN_ADDR, add config.yaml volume |
| `config.example.yaml` | New file: documented example |

## Future Considerations

- **Config validation:** A follow-up could add validation (e.g. port range checks, URL format validation) in `AppConfig.Validate()` called after Load.
- **Hot reload:** Not in scope, but the design (single struct, no global state) does not preclude it.
- **Environment variable override:** A later addition could allow env vars to selectively override YAML fields (useful in Docker/Kubernetes without mounting a config file).
