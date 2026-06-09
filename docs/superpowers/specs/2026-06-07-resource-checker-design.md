# Resource Availability Checker

**Date:** 2026-06-07
**Status:** Draft
**Branch:** feat/resource-checker

## Motivation

When an ISP blocks access to specific services (e.g., YouTube, Telegram in Russia),
users need to verify that their proxy nodes can reach those blocked resources.
This feature adds configurable resource availability checking through proxy tunnels,
with results stored in a local SQLite database.

## 1. Refactor: Extract TempRuntime to tunnelrunner

### Current State

The `speedtest` package contains the `TempRuntime` interface and its implementations
(Docker, native) for launching temporary sing-box proxy instances.

### Target

Move tunnel management into a shared package:

```
internal/pkg/tunnelrunner/
├── runner.go       — interface Runner (was TempRuntime)
├── docker.go       — Docker implementation
└── native.go       — Native (exec.Command) implementation
```

### Changes

| File (before) | File (after) |
|---|---|
| `speedtest/runtimer.go` | `tunnelrunner/runner.go` |
| `speedtest/runtimer_docker.go` | `tunnelrunner/docker.go` |
| `speedtest/runtimer_native.go` | `tunnelrunner/native.go` |

- Interface: `TempRuntime` → `Runner`
- Constructor: `NewTempRuntime(cfg)` → `NewRunner(cfg)`
- Methods unchanged: `StartTemp`, `StopTemp`, `WaitTempReady`, `GetTempLogs`
- `speedtest/interfaces.go` retains only `NodeProvider`, `SpeedTestResultSaver`
- `speedtest/service.go` imports `tunnelrunner` and uses `tunnelrunner.Runner`

## 2. New Package: resourcecheck

### 2.1 Configuration File

`DATA_DIR/resources.yaml` — defines resources to check.

```yaml
resources:
  - name: youtube
    url: https://www.youtube.com
    type: http
  - name: telegram
    url: https://telegram.org
    type: http
  - name: google-dns
    url: 8.8.8.8
    type: tcp
    port: 443
```

- Loaded via `gopkg.in/yaml.v3`
- If file doesn't exist — empty resource list (no error)
- Reloadable via API without restart

### 2.2 SQLite Database

**File:** `DATA_DIR/resource_checks.db`

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS tags (
    tag TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS check_results (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    resource    TEXT NOT NULL,
    tag         TEXT NOT NULL,
    status      TEXT NOT NULL,      -- 'ok' | 'timeout' | 'error'
    latency_ms  INTEGER,
    http_code   INTEGER,
    error       TEXT,
    checked_at  TEXT NOT NULL,      -- ISO 8601
    FOREIGN KEY (tag) REFERENCES tags(tag)
);

CREATE INDEX IF NOT EXISTS idx_results_tag_resource ON check_results(tag, resource);
CREATE INDEX IF NOT EXISTS idx_results_resource ON check_results(resource);
CREATE INDEX IF NOT EXISTS idx_results_checked_at ON check_results(checked_at);

CREATE VIEW IF NOT EXISTS latest_results AS
SELECT cr.* FROM check_results cr
INNER JOIN (
    SELECT resource, tag, MAX(id) as max_id
    FROM check_results
    GROUP BY resource, tag
) latest ON cr.id = latest.max_id;
```

### 2.3 Go Models

```go
type ResourceConfig struct {
    Name string `yaml:"name"`
    URL  string `yaml:"url"`
    Type string `yaml:"type"`       // "http" | "tcp"
    Port int    `yaml:"port,omitempty"`
}

type CheckResult struct {
    ID        int64  `json:"id" db:"id"`
    Resource  string `json:"resource" db:"resource"`
    Tag       string `json:"tag" db:"tag"`
    Status    string `json:"status" db:"status"`
    LatencyMs int64  `json:"latency_ms" db:"latency_ms"`
    HTTPCode  int    `json:"http_code,omitempty" db:"http_code"`
    Error     string `json:"error,omitempty" db:"error"`
    CheckedAt string `json:"checked_at" db:"checked_at"`
}
```

### 2.4 Check Mechanism

Per node:

1. `tunnelrunner.Runner.StartTemp()` — start sing-box tunnel on local port
2. `WaitTempReady()` — wait for port to accept connections
3. For each resource:
   - **HTTP type:** GET via `http.Client{Proxy: proxyURL}` — success if status is 2xx/3xx/4xx (request reached server)
   - **TCP type:** `net.Dial` through proxy — success if connection established
   - Save `CheckResult` to SQLite
4. `tunnelrunner.Runner.StopTemp()` — stop tunnel

### 2.5 Background Scheduler

- `POST /api/resourcecheck/schedule {"interval_sec": N}` — periodic check of all nodes
- `{"interval_sec": 0}` — stop background checks
- Inactive by default on service start

### 2.6 API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/resourcecheck/resources` | List resources from config |
| `GET` | `/api/resourcecheck/results` | All latest results (latest_results view) |
| `GET` | `/api/resourcecheck/results/:tag` | Latest results for a specific node |
| `GET` | `/api/resourcecheck/history/:resource/:tag` | Check history for (resource, tag) |
| `POST` | `/api/resourcecheck/run` | Run check: `{}` (all), `{"tag":"..."}`, `{"subscription_id":"..."}` |
| `POST` | `/api/resourcecheck/stop` | Stop running check |
| `POST` | `/api/resourcecheck/schedule` | Configure background interval |
| `GET` | `/api/resourcecheck/status` | Current status and progress |
| `POST` | `/api/resourcecheck/reload` | Reload resources.yaml |

### 2.7 Package Structure

```
server/internal/resourcecheck/
├── models.go          — ResourceConfig, CheckResult, Status, request/response types
├── store.go           — SQLite: InitDB, SaveResult, GetLatestResults, GetHistory
├── checker.go         — HTTP/TCP checks through tunnelrunner.Runner
├── service.go         — Orchestrator: RunOnce, RunForTags, schedule, stop
├── handler.go         — Gin handlers
├── register.go        — RegisterRoutes
└── interfaces.go      — Runner, NodeProvider interfaces
```

## 3. Changes to main.go

```go
import (
    "singbox-config-service/internal/pkg/tunnelrunner"
    "singbox-config-service/internal/resourcecheck"
)

// Shared tunnel runner
tr := tunnelrunner.NewRunner(cfg)

// Speedtest uses tunnelrunner
speedtestSvc := speedtest.NewService(tr, cfg)

// Resource checker uses the same tunnel runner
rcSvc := resourcecheck.NewService(tr, cfg.GetDataDir(), subSvc)

// Route registration
rcGroup := api.Group("/resourcecheck")
resourcecheckHandler.RegisterRoutes(rcGroup)
```

## 4. Testing

- **tunnelrunner:** migrate existing tests if any
- **speedtest:** update imports, verify `go test ./...` passes
- **resourcecheck/store_test.go:** in-memory SQLite, CRUD operations, latest_results view
- **resourcecheck/checker_test.go:** mock tunnelrunner.Runner, test HTTP/TCP check logic
