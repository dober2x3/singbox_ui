# Config Dependency Inversion Design

**Date:** 2026-06-27
**Status:** Specified

## Problem

The `pkg/config` package currently depends on three domain packages:

```
pkg/config
  ├── imports → prober        (prober.Config)
  ├── imports → scheduler     (scheduler.Config)
  └── imports → subscription  (subscription.Config)
  └── duplicates SpeedtestConfig inline (because speedtest imports config → cycle)
```

This is an inverted dependency: infrastructure/config should NOT depend on domain logic. Domain packages should define and parse their own configuration. Additionally:

- `SpeedtestConfig` is duplicated between `pkg/config` and `speedtest` package due to circular import prevention
- `defaultAppConfig()` needs to know about all domain configs, creating coupling
- `mergeDefaults()` manually handles defaults for each domain's fields in config
- Adding a new domain config section requires modifying `pkg/config`
- Default `DataDir` auto-detection logic (go.mod check) is fragile and non-obvious

## Solution

### 1. Extract domain configs to their packages

Each domain package gains a `ParseConfig(*yaml.Node) (Config, error)` function that:
- Returns `DefaultConfig()` if the node is nil/zero (section absent from YAML)
- Decodes the node into the config struct (preserving defaults for absent fields)
- Applys defaults for zero-valued fields after decode

### 2. Simplify `AppConfig` to use `yaml.Node`

`AppConfig` replaces typed domain config fields with `yaml.Node`:

```go
type AppConfig struct {
    Server       ServerConfig `yaml:"server"`
    Prober       yaml.Node    `yaml:"prober"`
    Speedtest    yaml.Node    `yaml:"speedtest"`
    Scheduler    yaml.Node    `yaml:"scheduler"`
    Subscription yaml.Node    `yaml:"subscription"`
}
```

### 3. Remove duplicated `SpeedtestConfig`

The inline `SpeedtestConfig` in `config.go` is deleted. `speedtest` already has its own `Config` struct with `DefaultConfig()`.

### 4. Change default DataDir

Replace auto-detection (go.mod check) with `$HOME/singbox_ui` using `os/user.Current().HomeDir`.

## Changes by file

### `server/internal/pkg/config/config.go`

**Remove imports:**
- `"singbox-config-service/internal/prober"`
- `"singbox-config-service/internal/scheduler"`
- `"singbox-config-service/internal/subscription"`

**Remove:**
- `SpeedtestConfig` struct
- `defaultSpeedtestConfig()` function
- `defaultAppConfig()` function
- `mergeDefaults()` function

**Add:**
- `defaultDataDir()` — returns `$HOME/singbox_ui` via `os/user`
- Import `"os/user"`

**Modify:**
- `AppConfig` — domain fields become `yaml.Node`
- `Load()` — simplified, no pre-population of defaults for domain sections
- `Init()` — uses `defaultDataDir()` instead of go.mod-based detection

**Keep unchanged:**
- `ServerConfig`
- `GetDataDir()`, `GetListenAddr()`, `GetSingboxDir()`, `GetSingboxBinPath()`, `ResolveHostConfigDir()`

### `server/internal/prober/models.go`

Add:
```go
func ParseConfig(node *yaml.Node) (Config, error) {
    cfg := DefaultConfig()
    if node == nil || node.Kind == 0 { return cfg, nil }
    if err := node.Decode(&cfg); err != nil { return Config{}, err }
    def := DefaultConfig()
    if cfg.Interval == 0 { cfg.Interval = def.Interval }
    if cfg.Timeout == 0 { cfg.Timeout = def.Timeout }
    if cfg.Concurrent == 0 { cfg.Concurrent = def.Concurrent }
    if cfg.MaxResults == 0 { cfg.MaxResults = def.MaxResults }
    if cfg.MaxRetries == 0 { cfg.MaxRetries = def.MaxRetries }
    return cfg, nil
}
```

Add import `"gopkg.in/yaml.v3"`.

### `server/internal/scheduler/models.go`

Add:
```go
func ParseConfig(node *yaml.Node) (Config, error) {
    cfg := DefaultConfig()
    if node == nil || node.Kind == 0 { return cfg, nil }
    if err := node.Decode(&cfg); err != nil { return Config{}, err }
    if cfg.Interval == 0 { cfg.Interval = DefaultConfig().Interval }
    return cfg, nil
}
```

Add import `"gopkg.in/yaml.v3"`.

### `server/internal/subscription/models.go`

Add:
```go
func ParseConfig(node *yaml.Node) (Config, error) {
    cfg := DefaultConfig()
    if node == nil || node.Kind == 0 { return cfg, nil }
    if err := node.Decode(&cfg); err != nil { return Config{}, err }
    return cfg, nil
}
```

Add import `"gopkg.in/yaml.v3"`.

### `server/internal/speedtest/models.go`

Add:
```go
func ParseConfig(node *yaml.Node) (Config, error) {
    cfg := DefaultConfig()
    if node == nil || node.Kind == 0 { return cfg, nil }
    if err := node.Decode(&cfg); err != nil { return Config{}, err }
    def := DefaultConfig()
    if cfg.LatencyURL == "" { cfg.LatencyURL = def.LatencyURL }
    if cfg.DownloadURL == "" { cfg.DownloadURL = def.DownloadURL }
    if cfg.Duration == 0 { cfg.Duration = def.Duration }
    return cfg, nil
}
```

Add import `"gopkg.in/yaml.v3"`.

### `server/main.go`

Replace typed config field access with ParseConfig calls:

```go
// Before:
subSvc := subscription.NewService(subStore, cfg.Subscription)
proberSvc := prober.NewService(cfg.Prober, cfg.GetDataDir(), subSvc)
sched := scheduler.New(subSvc, nil, cfg.Scheduler)
speedtestSvc := speedtest.NewService(tr, cfg, speedtest.Config{
    LatencyURL:  cfg.Speedtest.LatencyURL,
    DownloadURL: cfg.Speedtest.DownloadURL,
    Duration:    cfg.Speedtest.Duration,
})

// After:
subCfg, _ := subscription.ParseConfig(&cfg.Subscription)
proberCfg, _ := prober.ParseConfig(&cfg.Prober)
schedCfg, _ := scheduler.ParseConfig(&cfg.Scheduler)
speedCfg, _ := speedtest.ParseConfig(&cfg.Speedtest)

subSvc := subscription.NewService(subStore, subCfg)
proberSvc := prober.NewService(proberCfg, cfg.GetDataDir(), subSvc)
sched := scheduler.New(subSvc, nil, schedCfg)
speedtestSvc := speedtest.NewService(tr, cfg, speedCfg)
```

### `server/internal/pkg/config/config_test.go`

Update tests to use domain ParseConfig where they check sub-config field values:

- `TestLoadConfig_partial` — use `prober.ParseConfig(&cfg.Prober).Interval`
- `TestLoadConfig_full` — use `prober.ParseConfig` and `speedtest.ParseConfig`
- `TestInit_defaultPath` — update expected default path logic
- Remove tests that relied on `defaultAppConfig()` internals

## Dependency graph after

```
pkg/config (no domain imports)
  │   ServerConfig, yaml.Node for sub-sections
  │   Init(), Load(), GetDataDir(), ...
  │
  ├── imported by → main.go
  ├── imported by → singbox/* (runtime, service)
  ├── imported by → tunnelrunner
  ├── imported by → speedtest/service.go
  └── imported by → resourcecheck/checker.go

Domain packages (each owns its config parsing):
  prober/models.go        ← ParseConfig(&yaml.Node)
  scheduler/models.go     ← ParseConfig(&yaml.Node)
  subscription/models.go  ← ParseConfig(&yaml.Node)
  speedtest/models.go     ← ParseConfig(&yaml.Node)
```

## Testing

- `config_test.go`: update to reflect simplified Load(); verify yaml.Node round-trips through ParseConfig
- Domain `ParseConfig`: test with nil node, zero node, and full YAML node to verify defaults merge correctly
- `main.go`: no tests, verify compilation only
