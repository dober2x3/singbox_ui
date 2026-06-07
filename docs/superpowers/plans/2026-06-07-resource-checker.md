# Resource Availability Checker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a configurable resource availability checker that tests whether blocked services (YouTube, Telegram, etc.) are reachable through proxy nodes, stores results in SQLite, and exposes via API.

**Architecture:** Extract tunnel management from speedtest into a shared `tunnelrunner` package. Create a new `resourcecheck` package with SQLite storage (sqlx), YAML-based resource config, HTTP/TCP checkers using tunnelrunner, and a background scheduler. Wire into main.go.

**Tech Stack:** Go 1.24, Gin, sqlx + modernc.org/sqlite (pure Go), gopkg.in/yaml.v3 (already in go.mod)

**Plan location:** `docs/superpowers/plans/2026-06-07-resource-checker.md`

---

## File Structure

### New files created:
```
internal/pkg/tunnelrunner/
├── runner.go              — interface Runner (moved from speedtest)

internal/resourcecheck/
├── models.go              — ResourceConfig, CheckResult, status/request/response types
├── store.go               — SQLite DB: InitDB, SaveResult, GetLatestResults, GetHistory, GetResultsForTag
├── store_test.go          — In-memory SQLite tests for store
├── checker.go             — HTTP/TCP resource checking through tunnelrunner
├── service.go             — Orchestrator: run, stop, schedule
├── service_test.go        — Tests for service orchestration
├── handler.go             — Gin HTTP handlers
├── handler_test.go        — Tests for HTTP handlers
├── register.go            — RegisterRoutes
└── interfaces.go          — Runner, NodeProvider interfaces
```

### Modified files:
```
server/internal/speedtest/
├── runtimer.go            — REMOVED (moved to tunnelrunner/runner.go)
├── runtimer_docker.go     — REMOVED (moved to tunnelrunner/docker.go)
├── runtimer_native.go     — REMOVED (moved to tunnelrunner/native.go)
├── interfaces.go          — REMOVED (types split between packages)
├── service.go             — Import tunnelrunner; use tunnelrunner.Runner
└── service_test.go        — Update mockTempRuntime → mock implements tunnelrunner.Runner

server/main.go             — Wire resourcecheck service + routes

server/go.mod              — Add github.com/jmoiron/sqlx, modernc.org/sqlite
```

---

## Task Breakdown

### Task 1: Create tunnelrunner package

**Files:**
- Create: `server/internal/pkg/tunnelrunner/runner.go`

- [ ] **Step 1: Create `runner.go` with Runner interface**

```go
// Package tunnelrunner manages temporary sing-box proxy instances (tunnels)
// used for speed tests and resource availability checks.
package tunnelrunner

import (
	"context"
	"time"
)

// Runner manages the lifecycle of a temporary proxy (tunnel) instance.
type Runner interface {
	// StartTemp launches a temporary proxy from the given config and returns an instance ID.
	StartTemp(ctx context.Context, configPath string) (string, error)
	// StopTemp stops a running temporary proxy instance.
	StopTemp(ctx context.Context, id string) error
	// WaitTempReady blocks until the proxy is accepting connections on the given port.
	WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error
	// GetTempLogs returns the logs of a temporary proxy instance.
	GetTempLogs(ctx context.Context, id string) string
}
```

- [ ] **Step 2: Verify build passes**

Run: `go build ./internal/pkg/tunnelrunner/...`
Expected: success (no implementations yet, but empty package with interface compiles)

- [ ] **Step 3: Commit**

```bash
git add server/internal/pkg/tunnelrunner/
git commit -m "refactor: create tunnelrunner package with Runner interface"
```

---

### Task 2: Move Docker/Native implementations to tunnelrunner

**Files:**
- Create: `server/internal/pkg/tunnelrunner/docker.go`
- Create: `server/internal/pkg/tunnelrunner/native.go`

- [ ] **Step 1: Create `docker.go`** — copy from `speedtest/runtimer_docker.go`, rename `TempRuntime` → `Runner`, `NewTempRuntime` → `NewRunner`

Content of `docker.go` (verbatim copy from speedtest with rename):

```go
package tunnelrunner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"singbox-config-service/internal/pkg/config"
)

type dockerRunner struct {
	cfg *config.Config
}

// NewRunner creates a new Docker-based Runner.
func NewRunner(cfg *config.Config) Runner {
	return &dockerRunner{cfg: cfg}
}

func (d *dockerRunner) StartTemp(ctx context.Context, configPath string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	containerName := fmt.Sprintf("speedtest-%d", time.Now().UnixNano())

	hostConfigPath, err := d.cfg.ResolveHostConfigDir(filepath.Dir(configPath))
	if err != nil {
		return "", fmt.Errorf("resolve host config dir: %w", err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "ghcr.io/sagernet/sing-box:latest",
		Cmd:   []string{"run", "-c", "/etc/sing-box/config.json"},
		Labels: map[string]string{
			"managed-by": "singbox-config-service",
		},
	}, &container.HostConfig{
		Binds:           []string{fmt.Sprintf("%s:/etc/sing-box:ro", hostConfigPath)},
		NetworkMode:     container.NetworkMode("host"),
		AutoRemove:      true,
	}, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("container create: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("container start: %w", err)
	}

	return resp.ID, nil
}

func (d *dockerRunner) StopTemp(ctx context.Context, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	return cli.ContainerStop(ctx, id, container.StopOptions{})
}

func (d *dockerRunner) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// Use raw TCP dial — we're testing connectivity via Docker host network mode
		conn, err := (&net.Dialer{Timeout: 500 * time.Millisecond}).DialContext(ctx, "tcp", addr)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %d", port)
}

func (d *dockerRunner) GetTempLogs(ctx context.Context, id string) string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Sprintf("failed to create docker client: %v", err)
	}
	defer cli.Close()

	reader, err := cli.ContainerLogs(ctx, id, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return fmt.Sprintf("failed to get logs: %v", err)
	}
	defer reader.Close()

	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, reader)
	if err != nil {
		return fmt.Sprintf("failed to read logs: %v", err)
	}
	if stderr.Len() > 0 {
		return strings.TrimSpace(stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}
```

Wait — the Docker file uses `net` but doesn't import it. Let me adjust — it imports `"net"` and uses `net.Dialer`. Also the original had `pickFreePort` in speedtest — the Docker implementation didn't have that. Let me check the actual original files.

Actually, I'm writing the plan, not the implementation. Let me write steps that tell the implementer what to do, with enough code for clarity but not necessarily the full file content.

Let me restructure to be clearer. For file creation tasks, I'll reference what to copy/paste rather than including the entire code in the plan.

- [ ] **Step 1: Move `speedtest/runtimer_docker.go` → `tunnelrunner/docker.go`**

Copy `server/internal/speedtest/runtimer_docker.go` to `server/internal/pkg/tunnelrunner/docker.go`.
- Rename `type tempRuntime` → `type dockerRunner`
- Rename `func NewTempRuntime` → `func NewRunner`
- Change package from `speedtest` to `tunnelrunner`
- Change the `cfg` type import to `singbox-config-service/internal/pkg/config`
- Add `"net"` import (used in `WaitTempReady`)

- [ ] **Step 2: Move `speedtest/runtimer_native.go` → `tunnelrunner/native.go`**

Copy `server/internal/speedtest/runtimer_native.go` to `server/internal/pkg/tunnelrunner/native.go`.
- Rename `type tempRuntime` → `type nativeRunner`
- Rename `func NewTempRuntime` → `func NewRunner` (wait — both docker and native need separate constructors)
- Actually, the current code has a single `NewTempRuntime(cfg)` that decides which implementation to use. Let me check...

Actually, looking at the original code again:

The original `speedtest/runtimer.go` has `TempRuntime` interface.
`speedtest/runtimer_docker.go` has `type tempRuntime struct` implementing it.
`speedtest/runtimer_native.go` has `type tempRuntime struct` implementing it.

There's a build tag selection: `runtimer_native.go` has `//go:build native` and `runtimer_docker.go` is the default (no build tag or `//go:build !native`).

Each file has its own `NewTempRuntime` function.

So for tunnelrunner:
- `docker.go` — default implementation, `//go:build !native` — has `type dockerRunner struct` and `NewRunner(cfg)`
- `native.go` — `//go:build native` — has `type nativeRunner struct` and `NewRunner(cfg)`

But the constructor names would conflict if both are compiled. So we need to keep build tags or have separate constructors. Actually that's already how it works — build tags ensure only one implementation is compiled at a time, so both can have `NewRunner(cfg)`.

Let me write the plan correctly. I'll use the approach of just moving the files with necessary renames.

- [ ] **Step 3: Verify build passes**

Run: `go build ./internal/pkg/tunnelrunner/...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add server/internal/pkg/tunnelrunner/
git commit -m "refactor: move Runner implementations to tunnelrunner package"
```

---

### Task 3: Update speedtest to use tunnelrunner

**Files:**
- Modify: `server/internal/speedtest/service.go`
- Modify: `server/internal/speedtest/service_test.go`
- Delete: `server/internal/speedtest/runtimer.go`
- Delete: `server/internal/speedtest/runtimer_docker.go`
- Delete: `server/internal/speedtest/runtimer_native.go`
- Create: `server/internal/speedtest/interfaces.go` (new, with only NodeProvider + SpeedTestResultSaver)

- [ ] **Step 1: Delete old runtime files**

Remove:
```bash
rm server/internal/speedtest/runtimer.go
rm server/internal/speedtest/runtimer_docker.go
rm server/internal/speedtest/runtimer_native.go
```

- [ ] **Step 2: Create new `interfaces.go`** in speedtest with only NodeProvider and SpeedTestResultSaver

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

- [ ] **Step 3: Update `service.go`** — replace `TempRuntime` import/usage with `tunnelrunner.Runner`

In `server/internal/speedtest/service.go`:
- Add import: `"singbox-config-service/internal/pkg/tunnelrunner"`
- Remove import: `"singbox-config-service/internal/pkg/config"` (keep it — still used for config)
- Change `Service.tempRuntime` field type: `TempRuntime` → `tunnelrunner.Runner`
- Change `NewService` parameter type: `tempRuntime TempRuntime` → `runner tunnelrunner.Runner`
- Inside `NewService`, change field assignment: `tempRuntime: tempRuntime` → `tempRuntime: runner` (or better rename the field)

Actually, let me think about what we should do about the Service struct. The field `tempRuntime` should probably be renamed to `runner` for clarity, but that's cosmetic. Let me just change the type.

```go
type Service struct {
	runner       tunnelrunner.Runner
	cfg          *config.Config
	nodeProvider NodeProvider
	resultSaver  SpeedTestResultSaver
	state        *SpeedTestState
	mu           sync.Mutex
	cancel       context.CancelFunc
}

func NewService(runner tunnelrunner.Runner, cfg *config.Config) *Service {
	return &Service{
		runner: runner,
		cfg:    cfg,
		state:  &SpeedTestState{},
	}
}
```

Update all references to `s.tempRuntime` → `s.runner` in service.go:
- `s.tempRuntime.StartTemp(...)` → `s.runner.StartTemp(...)`
- `s.tempRuntime.StopTemp(...)` → `s.runner.StopTemp(...)`
- `s.tempRuntime.WaitTempReady(...)` → `s.runner.WaitTempReady(...)`
- `s.tempRuntime.GetTempLogs(...)` → `s.runner.GetTempLogs(...)`

- [ ] **Step 4: Update `service_test.go`** — make mockTempRuntime implement `tunnelrunner.Runner`

In `service_test.go`:
- Add import: `"singbox-config-service/internal/pkg/tunnelrunner"`
- The mock already has all the right methods — just need to ensure the mock struct implements `tunnelrunner.Runner`
- Change `newMockTempRuntime()` returns `*mockTempRuntime` (no change needed structurally)
- The mock methods already match the interface — this should compile once the interface changes are done

- [ ] **Step 5: Run tests to verify**

Run: `go test ./internal/speedtest/...`
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add server/internal/speedtest/
git commit -m "refactor: speedtest uses tunnelrunner.Runner instead of local TempRuntime"
```

---

### Task 4: Add SQLite dependencies

**Files:**
- Modify: `server/go.mod`
- Modify: `server/go.sum`

- [ ] **Step 1: Add sqlx and modernc.org/sqlite**

```bash
cd server && go get github.com/jmoiron/sqlx modernc.org/sqlite
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: success

- [ ] **Step 3: Commit**

```bash
git add server/go.mod server/go.sum
git commit -m "deps: add sqlx and modernc.org/sqlite dependencies"
```

---

### Task 5: Create resourcecheck — models and store

**Files:**
- Create: `server/internal/resourcecheck/models.go`
- Create: `server/internal/resourcecheck/store.go`
- Create: `server/internal/resourcecheck/store_test.go`

- [ ] **Step 1: Create `models.go`**

```go
package resourcecheck

import "time"

// ResourceConfig defines a single resource to check, loaded from resources.yaml.
type ResourceConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Type string `yaml:"type"`       // "http" | "tcp"
	Port int    `yaml:"port,omitempty"`
}

// ResourceConfigFile is the root YAML structure.
type ResourceConfigFile struct {
	Resources []ResourceConfig `yaml:"resources"`
}

// LoadResources reads resources from a YAML file. Returns empty slice if file missing.
func LoadResources(path string) ([]ResourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg ResourceConfigFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.Resources, nil
}

// CheckResult represents a single resource check result persisted in SQLite.
type CheckResult struct {
	ID        int64  `json:"id" db:"id"`
	Resource  string `json:"resource" db:"resource"`
	Tag       string `json:"tag" db:"tag"`
	Status    string `json:"status" db:"status"`    // "ok" | "timeout" | "error"
	LatencyMs int64  `json:"latency_ms" db:"latency_ms"`
	HTTPCode  int    `json:"http_code,omitempty" db:"http_code"`
	Error     string `json:"error,omitempty" db:"error"`
	CheckedAt string `json:"checked_at" db:"checked_at"` // ISO 8601
}

// CheckStatus tracks progress of a running check operation.
type CheckStatus struct {
	Running          bool   `json:"running"`
	Tag              string `json:"tag,omitempty"`
	Resource         string `json:"resource,omitempty"`
	Progress         int    `json:"progress,omitempty"`          // 0-100
	TotalNodes       int    `json:"total_nodes,omitempty"`
	CompletedNodes   int    `json:"completed_nodes,omitempty"`
	TotalChecks      int    `json:"total_checks,omitempty"`      // nodes × resources
	CompletedChecks  int    `json:"completed_checks,omitempty"`
	Status           string `json:"status,omitempty"`             // "idle" | "running" | "completed"
}

// RunRequest is the body for POST /api/resourcecheck/run.
type RunRequest struct {
	Tag            string `json:"tag,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
}

// ScheduleRequest is the body for POST /api/resourcecheck/schedule.
type ScheduleRequest struct {
	IntervalSec int `json:"interval_sec"`
}
```

- [ ] **Step 2: Create `store.go`**

```go
package resourcecheck

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Store persists check results to SQLite.
type Store struct {
	db *sqlx.DB
}

// NewStore opens (or creates) the SQLite database and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func migrate(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tags (
		tag TEXT PRIMARY KEY
	);
	CREATE TABLE IF NOT EXISTS check_results (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		resource    TEXT NOT NULL,
		tag         TEXT NOT NULL,
		status      TEXT NOT NULL,
		latency_ms  INTEGER,
		http_code   INTEGER,
		error       TEXT,
		checked_at  TEXT NOT NULL,
		FOREIGN KEY (tag) REFERENCES tags(tag)
	);
	CREATE INDEX IF NOT EXISTS idx_results_tag_resource ON check_results(tag, resource);
	CREATE INDEX IF NOT EXISTS idx_results_resource ON check_results(resource);
	CREATE INDEX IF NOT EXISTS idx_results_checked_at ON check_results(checked_at);
	`
	_, err := db.Exec(schema)
	return err
}

// SaveResult inserts a check result and ensures the tag exists.
func (s *Store) SaveResult(r CheckResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Upsert tag
	_, err = tx.Exec("INSERT OR IGNORE INTO tags (tag) VALUES (?)", r.Tag)
	if err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}

	// Insert result
	_, err = tx.Exec(
		`INSERT INTO check_results (resource, tag, status, latency_ms, http_code, error, checked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.Resource, r.Tag, r.Status, r.LatencyMs, r.HTTPCode, r.Error, r.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("insert result: %w", err)
	}

	return tx.Commit()
}

// GetLatestResults returns the most recent result for each (resource, tag) pair.
func (s *Store) GetLatestResults() ([]CheckResult, error) {
	var results []CheckResult
	err := s.db.Select(&results, `SELECT * FROM latest_results ORDER BY tag, resource`)
	return results, err
}

// GetResultsForTag returns the latest results for a specific node tag.
func (s *Store) GetResultsForTag(tag string) ([]CheckResult, error) {
	var results []CheckResult
	err := s.db.Select(&results,
		`SELECT * FROM latest_results WHERE tag = ? ORDER BY resource`, tag)
	return results, err
}

// GetHistory returns check history for a (resource, tag) pair, newest first.
func (s *Store) GetHistory(resource, tag string, limit int) ([]CheckResult, error) {
	if limit <= 0 {
		limit = 50
	}
	var results []CheckResult
	err := s.db.Select(&results,
		`SELECT * FROM check_results WHERE resource = ? AND tag = ?
		 ORDER BY id DESC LIMIT ?`, resource, tag, limit)
	return results, err
}

// GetTags returns all known proxy node tags.
func (s *Store) GetTags() ([]string, error) {
	var tags []string
	err := s.db.Select(&tags, "SELECT tag FROM tags ORDER BY tag")
	return tags, err
}

// Close shuts down the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
```

- [ ] **Step 3: Create `store_test.go`**

```go
package resourcecheck

import (
	"testing"
	"time"
)

func TestStore_CreateAndQuery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	result := CheckResult{
		Resource:  "youtube",
		Tag:       "node-1",
		Status:    "ok",
		LatencyMs: 150,
		HTTPCode:  200,
		CheckedAt: now,
	}

	if err := store.SaveResult(result); err != nil {
		t.Fatalf("SaveResult() error = %v", err)
	}

	// Query latest results
	results, err := store.GetLatestResults()
	if err != nil {
		t.Fatalf("GetLatestResults() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Resource != "youtube" {
		t.Errorf("expected resource youtube, got %s", results[0].Resource)
	}
	if results[0].Status != "ok" {
		t.Errorf("expected status ok, got %s", results[0].Status)
	}

	// Save a newer result for same resource+tag
	result2 := CheckResult{
		Resource:  "youtube",
		Tag:       "node-1",
		Status:    "timeout",
		LatencyMs: -1,
		CheckedAt: time.Now().UTC().Add(time.Second).Format(time.RFC3339),
	}
	if err := store.SaveResult(result2); err != nil {
		t.Fatalf("SaveResult() error = %v", err)
	}

	// Latest should return only the newest
	results, err = store.GetLatestResults()
	if err != nil {
		t.Fatalf("GetLatestResults() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 latest result, got %d", len(results))
	}
	if results[0].Status != "timeout" {
		t.Errorf("expected latest status timeout, got %s", results[0].Status)
	}

	// History should return both
	history, err := store.GetHistory("youtube", "node-1", 10)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
}

func TestStore_GetResultsForTag(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	store.SaveResult(CheckResult{Resource: "youtube", Tag: "node-1", Status: "ok", LatencyMs: 100, CheckedAt: now})
	store.SaveResult(CheckResult{Resource: "telegram", Tag: "node-1", Status: "ok", LatencyMs: 50, CheckedAt: now})
	store.SaveResult(CheckResult{Resource: "youtube", Tag: "node-2", Status: "timeout", LatencyMs: -1, CheckedAt: now})

	results, err := store.GetResultsForTag("node-1")
	if err != nil {
		t.Fatalf("GetResultsForTag() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for node-1, got %d", len(results))
	}

	tags, err := store.GetTags()
	if err != nil {
		t.Fatalf("GetTags() error = %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
}

func TestStore_EmptyDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	results, err := store.GetLatestResults()
	if err != nil {
		t.Fatalf("GetLatestResults() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results in empty db, got %d", len(results))
	}
}
```

- [ ] **Step 4: Run store tests**

Run: `go test ./internal/resourcecheck/... -run TestStore`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/resourcecheck/models.go server/internal/resourcecheck/store.go server/internal/resourcecheck/store_test.go
git commit -m "feat(resourcecheck): add models and SQLite store"
```

---

### Task 6: Create resourcecheck — interfaces and checker

**Files:**
- Create: `server/internal/resourcecheck/interfaces.go`
- Create: `server/internal/resourcecheck/checker.go`
- Create: `server/internal/resourcecheck/checker_test.go`

- [ ] **Step 1: Create `interfaces.go`**

```go
package resourcecheck

import (
	"context"
	"time"

	"singbox-config-service/internal/pkg/types"
)

// Runner is the tunnel lifecycle manager (aliased for convenience).
type Runner interface {
	StartTemp(ctx context.Context, configPath string) (string, error)
	StopTemp(ctx context.Context, id string) error
	WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error
	GetTempLogs(ctx context.Context, id string) string
}

// NodeProvider provides proxy nodes to check resources through.
type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}
```

- [ ] **Step 2: Create `checker.go`**

```go
package resourcecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

const (
	checkTimeout = 10 * time.Second
)

// Checker runs resource availability checks through proxy tunnels.
type Checker struct {
	runner Runner
	cfg    *config.Config
}

// NewChecker creates a Checker with the given Runner and Config.
func NewChecker(runner Runner, cfg *config.Config) *Checker {
	return &Checker{runner: runner, cfg: cfg}
}

// CheckNodeResources checks all resources through a single proxy node.
// Returns one CheckResult per resource.
func (c *Checker) CheckNodeResources(ctx context.Context, node *types.ProxyNode, resources []ResourceConfig) ([]CheckResult, error) {
	if node.Outbound == nil {
		return nil, fmt.Errorf("missing outbound for node %s", node.Name)
	}

	tag := nodeOutboundTag(node)
	port, err := pickFreePort()
	if err != nil {
		return nil, fmt.Errorf("pick port: %w", err)
	}

	// Build sing-box config and write to temp file
	cfg := buildCheckConfig(node, tag, port)
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(c.cfg.GetSingboxDir(), "resourcecheck")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(dir, fmt.Sprintf("config-%s.json", tag))
	if err := os.WriteFile(cfgPath, cfgBytes, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(cfgPath)

	// Start tunnel
	id, err := c.runner.StartTemp(ctx, cfgPath)
	if err != nil {
		return nil, fmt.Errorf("start tunnel: %w", err)
	}
	defer func() {
		_ = c.runner.StopTemp(ctx, id)
	}()

	if err := c.runner.WaitTempReady(ctx, id, port, 15*time.Second); err != nil {
		logs := c.runner.GetTempLogs(ctx, id)
		return nil, fmt.Errorf("tunnel not ready (port %d): %s", port, logs)
	}

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Check each resource
	results := make([]CheckResult, 0, len(resources))
	for _, res := range resources {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result := c.checkOne(ctx, proxyURL, tag, res)
		results = append(results, result)
	}

	return results, nil
}

// checkOne performs a single resource check through the proxy.
func (c *Checker) checkOne(ctx context.Context, proxyURL, tag string, res ResourceConfig) CheckResult {
	start := time.Now()

	switch res.Type {
	case "http":
		return c.checkHTTP(ctx, proxyURL, tag, res, start)
	case "tcp":
		return c.checkTCP(ctx, proxyURL, tag, res, start)
	default:
		return CheckResult{
			Resource:  res.Name,
			Tag:       tag,
			Status:    "error",
			Error:     fmt.Sprintf("unknown check type: %s", res.Type),
			CheckedAt: start.UTC().Format(time.RFC3339),
		}
	}
}

func (c *Checker) checkHTTP(ctx context.Context, proxyURL, tag string, res ResourceConfig, start time.Time) CheckResult {
	pu, _ := url.Parse(proxyURL)
	client := &http.Client{
		Timeout: checkTimeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(pu),
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: 8 * time.Second,
		},
	}
	defer client.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, "GET", res.URL, nil)
	if err != nil {
		return CheckResult{
			Resource: res.Name, Tag: tag, Status: "error",
			Error: err.Error(), CheckedAt: start.UTC().Format(time.RFC3339),
		}
	}

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return CheckResult{
			Resource: res.Name, Tag: tag, Status: "timeout",
			LatencyMs: latency, Error: err.Error(), CheckedAt: start.UTC().Format(time.RFC3339),
		}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	status := "ok"
	if resp.StatusCode >= 500 {
		status = "error"
	}

	return CheckResult{
		Resource:  res.Name,
		Tag:       tag,
		Status:    status,
		LatencyMs: latency,
		HTTPCode:  resp.StatusCode,
		CheckedAt: start.UTC().Format(time.RFC3339),
	}
}

func (c *Checker) checkTCP(ctx context.Context, proxyURL, tag string, res ResourceConfig, start time.Time) CheckResult {
	pu, _ := url.Parse(proxyURL)
	dialer := &net.Dialer{Timeout: checkTimeout}

	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(res.URL, fmt.Sprintf("%d", res.Port)))
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return CheckResult{
			Resource: res.Name, Tag: tag, Status: "timeout",
			LatencyMs: latency, Error: err.Error(), CheckedAt: start.UTC().Format(time.RFC3339),
		}
	}
	conn.Close()

	return CheckResult{
		Resource:  res.Name,
		Tag:       tag,
		Status:    "ok",
		LatencyMs: latency,
		CheckedAt: start.UTC().Format(time.RFC3339),
	}
}

// nodeOutboundTag extracts the tag from a node's outbound config or generates one.
func nodeOutboundTag(n *types.ProxyNode) string {
	if n.Outbound != nil {
		if t, ok := n.Outbound["tag"].(string); ok && t != "" {
			return t
		}
	}
	return fmt.Sprintf("%s-%s-%d", n.Protocol, n.Address, n.Port)
}

// pickFreePort finds a free TCP port.
func pickFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// buildCheckConfig creates a sing-box config for resource checking through a proxy node.
func buildCheckConfig(node *types.ProxyNode, tag string, port int) map[string]interface{} {
	outbound := make(map[string]interface{}, len(node.Outbound)+1)
	for k, v := range node.Outbound {
		outbound[k] = v
	}
	outbound["tag"] = tag

	return map[string]interface{}{
		"log": map[string]interface{}{"level": "warn"},
		"dns": map[string]interface{}{
			"servers": []map[string]interface{}{
				{"tag": "remote_dns", "type": "udp", "server": "8.8.8.8", "detour": tag},
				{"tag": "local_resolver", "type": "udp", "server": "1.1.1.1"},
			},
			"final":             "remote_dns",
			"independent_cache": true,
		},
		"inbounds": []map[string]interface{}{
			{
				"type":        "mixed",
				"tag":         "resourcecheck-in",
				"listen":      "127.0.0.1",
				"listen_port": port,
			},
		},
		"outbounds": []map[string]interface{}{
			outbound,
		},
		"route": map[string]interface{}{
			"rules":                   []interface{}{},
			"final":                   tag,
			"default_domain_resolver": "local_resolver",
		},
	}
}
```

- [ ] **Step 3: Create `checker_test.go`** (mock-based, tests parser logic without tunnel)

```go
package resourcecheck

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

// mockRunner implements Runner for testing.
type mockRunner struct {
	mu             sync.Mutex
	startCalled    bool
	stopCalled     bool
	startErr       error
	stopErr        error
	readyErr       error
	instanceID     string
}

func (m *mockRunner) StartTemp(ctx context.Context, configPath string) (string, error) {
	m.mu.Lock()
	m.startCalled = true
	id := m.instanceID
	err := m.startErr
	m.mu.Unlock()
	if id == "" {
		id = "mock-id"
	}
	return id, err
}

func (m *mockRunner) StopTemp(ctx context.Context, id string) error {
	m.mu.Lock()
	m.stopCalled = true
	err := m.stopErr
	m.mu.Unlock()
	return err
}

func (m *mockRunner) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	return m.readyErr
}

func (m *mockRunner) GetTempLogs(ctx context.Context, id string) string {
	return ""
}

func TestChecker_CheckNodeResources_MissingOutbound(t *testing.T) {
	cfg, _ := config.Init()
	checker := NewChecker(&mockRunner{}, cfg)
	node := &types.ProxyNode{Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443}

	_, err := checker.CheckNodeResources(context.Background(), node, nil)
	if err == nil {
		t.Fatal("expected error for missing outbound")
	}
}

func TestChecker_CheckNodeResources_TunnelStartFails(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}

	mock := &mockRunner{startErr: fmt.Errorf("start failed")}
	checker := NewChecker(mock, cfg)

	node := &types.ProxyNode{
		Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
		Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
	}

	_, err = checker.CheckNodeResources(context.Background(), node, []ResourceConfig{
		{Name: "youtube", URL: "https://www.youtube.com", Type: "http"},
	})
	if err == nil {
		t.Fatal("expected error when tunnel start fails")
	}
}

func TestNodeOutboundTag(t *testing.T) {
	node := &types.ProxyNode{
		Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
		Outbound: map[string]interface{}{"tag": "custom-tag"},
	}
	if tag := nodeOutboundTag(node); tag != "custom-tag" {
		t.Errorf("nodeOutboundTag() = %s, want custom-tag", tag)
	}
	node.Outbound = nil
	if tag := nodeOutboundTag(node); tag != "vmess-1.1.1.1-443" {
		t.Errorf("nodeOutboundTag() = %s, want vmess-1.1.1.1-443", tag)
	}
}

func TestBuildCheckConfig(t *testing.T) {
	node := &types.ProxyNode{
		Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
		Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
	}
	cfg := buildCheckConfig(node, "vmess-1_1_1_1-443", 10800)
	if cfg == nil {
		t.Fatal("buildCheckConfig returned nil")
	}
}

func TestPickFreePort(t *testing.T) {
	port, err := pickFreePort()
	if err != nil {
		t.Fatalf("pickFreePort() error = %v", err)
	}
	if port <= 0 {
		t.Errorf("port = %d, want > 0", port)
	}
}
```

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/resourcecheck/...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/resourcecheck/checker.go server/internal/resourcecheck/checker_test.go server/internal/resourcecheck/interfaces.go
git commit -m "feat(resourcecheck): add checker with HTTP/TCP resource checking"
```

---

### Task 7: Create resourcecheck — service layer

**Files:**
- Create: `server/internal/resourcecheck/service.go`
- Create: `server/internal/resourcecheck/service_test.go`

- [ ] **Step 1: Create `service.go`**

```go
package resourcecheck

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
	"singbox-config-service/internal/pkg/types"
)

// Service orchestrates resource availability checks.
type Service struct {
	checker      *Checker
	store        *Store
	nodeProvider NodeProvider
	config       ProberConfig

	resourcesPath string
	resources     []ResourceConfig
	resourcesMu   sync.RWMutex

	status     CheckStatus
	statusMu   sync.Mutex
	cancel     context.CancelFunc
	tickerStop chan struct{}
	wg         sync.WaitGroup
}

// ProberConfig holds configuration for the service.
type ProberConfig struct {
	ResourcesPath string // path to resources.yaml
	DBPath        string // path to SQLite database
}

// NewService creates a new resource check service.
func NewService(checker *Checker, nodeProvider NodeProvider, cfg ProberConfig) *Service {
	return &Service{
		checker:       checker,
		nodeProvider:  nodeProvider,
		config:        cfg,
		resourcesPath: cfg.ResourcesPath,
		status:        CheckStatus{Status: "idle"},
	}
}

// Init loads resources from YAML and opens the database.
func (s *Service) Init() error {
	// Open database
	store, err := NewStore(s.config.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	s.store = store

	// Load resources
	resources, err := LoadResources(s.resourcesPath)
	if err != nil {
		return fmt.Errorf("load resources: %w", err)
	}
	s.resourcesMu.Lock()
	s.resources = resources
	s.resourcesMu.Unlock()

	if len(resources) == 0 {
		log.Println("resourcecheck: no resources configured (resources.yaml missing or empty)")
	} else {
		log.Printf("resourcecheck: loaded %d resources from %s", len(resources), s.resourcesPath)
	}
	return nil
}

// ReloadResources reloads resources.yaml without restarting the service.
func (s *Service) ReloadResources() error {
	resources, err := LoadResources(s.resourcesPath)
	if err != nil {
		return err
	}
	s.resourcesMu.Lock()
	s.resources = resources
	s.resourcesMu.Unlock()
	log.Printf("resourcecheck: reloaded %d resources", len(resources))
	return nil
}

// GetResources returns the current resource list.
func (s *Service) GetResources() []ResourceConfig {
	s.resourcesMu.RLock()
	defer s.resourcesMu.RUnlock()
	result := make([]ResourceConfig, len(s.resources))
	copy(result, s.resources)
	return result
}

// RunAll checks all resources through all nodes.
func (s *Service) RunAll(ctx context.Context) error {
	allNodes := s.nodeProvider.GetAllNodes()
	if len(allNodes) == 0 {
		return fmt.Errorf("no nodes available")
	}
	s.resourcesMu.RLock()
	resources := make([]ResourceConfig, len(s.resources))
	copy(resources, s.resources)
	s.resourcesMu.RUnlock()
	if len(resources) == 0 {
		return fmt.Errorf("no resources configured")
	}
	return s.runChecks(ctx, allNodes, resources)
}

// RunForTag checks all resources through a specific node tag.
func (s *Service) RunForTag(ctx context.Context, tag string) error {
	allNodes := s.nodeProvider.GetAllNodes()
	var node *types.ProxyNode
	for i := range allNodes {
		if n := &allNodes[i]; nodeOutboundTag(n) == tag {
			node = n
			break
		}
	}
	if node == nil {
		return fmt.Errorf("node with tag %q not found", tag)
	}
	s.resourcesMu.RLock()
	resources := make([]ResourceConfig, len(s.resources))
	copy(resources, s.resources)
	s.resourcesMu.RUnlock()
	if len(resources) == 0 {
		return fmt.Errorf("no resources configured")
	}
	return s.runChecks(ctx, []types.ProxyNode{*node}, resources)
}

// runChecks iterates nodes and resources, performing checks.
func (s *Service) runChecks(ctx context.Context, nodes []types.ProxyNode, resources []ResourceConfig) error {
	s.setStatusRunning(len(nodes), len(resources))
	defer s.setStatusIdle()

	for i := range nodes {
		node := &nodes[i]
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tag := nodeOutboundTag(node)
		s.updateProgress(tag, "", i, len(nodes))

		results, err := s.checker.CheckNodeResources(ctx, node, resources)
		if err != nil {
			log.Printf("resourcecheck: check failed for node %s: %v", tag, err)
			continue
		}

		for _, r := range results {
			if err := s.store.SaveResult(r); err != nil {
				log.Printf("resourcecheck: save result error: %v", err)
			}
		}

		s.updateProgress(tag, "", i+1, len(nodes))
	}

	return nil
}

// GetLatestResults returns all latest results from the store.
func (s *Service) GetLatestResults() ([]CheckResult, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.GetLatestResults()
}

// GetResultsForTag returns latest results for a specific tag.
func (s *Service) GetResultsForTag(tag string) ([]CheckResult, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.GetResultsForTag(tag)
}

// GetHistory returns check history for a (resource, tag) pair.
func (s *Service) GetHistory(resource, tag string, limit int) ([]CheckResult, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.GetHistory(resource, tag, limit)
}

// GetStatus returns a copy of the current status.
func (s *Service) GetStatus() CheckStatus {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	return s.status
}

// Stop cancels a running check operation.
func (s *Service) Stop() {
	s.statusMu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.statusMu.Unlock()
}

// StartScheduler starts periodic checks with the given interval.
func (s *Service) StartScheduler(intervalSec int) {
	s.StopScheduler()
	if intervalSec <= 0 {
		return
	}

	s.tickerStop = make(chan struct{})
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-s.tickerStop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithCancel(context.Background())
				s.statusMu.Lock()
				s.cancel = cancel
				s.statusMu.Unlock()

				if err := s.RunAll(ctx); err != nil {
					log.Printf("resourcecheck: scheduled run error: %v", err)
				}
			}
		}
	}()
	log.Printf("resourcecheck: scheduler started with interval %ds", intervalSec)
}

// StopScheduler stops the periodic check scheduler.
func (s *Service) StopScheduler() {
	if s.tickerStop != nil {
		close(s.tickerStop)
		s.tickerStop = nil
	}
	s.wg.Wait()
}

// Close shuts down the service.
func (s *Service) Close() error {
	s.StopScheduler()
	s.Stop()
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

func (s *Service) setStatusRunning(totalNodes, totalChecks int) {
	s.statusMu.Lock()
	s.status = CheckStatus{
		Running:     true,
		Status:      "running",
		TotalNodes:  totalNodes,
		TotalChecks: totalChecks,
	}
	s.statusMu.Unlock()
}

func (s *Service) setStatusIdle() {
	s.statusMu.Lock()
	s.status.Running = false
	s.status.Status = "idle"
	s.statusMu.Unlock()
}

func (s *Service) updateProgress(tag, resource string, completed, total int) {
	s.statusMu.Lock()
	s.status.Tag = tag
	s.status.Resource = resource
	s.status.CompletedNodes = completed
	if total > 0 {
		s.status.Progress = completed * 100 / total
	}
	s.statusMu.Unlock()
}
```

- [ ] **Step 2: Create `service_test.go`**

```go
package resourcecheck

import (
	"context"
	"testing"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

type mockNodeProvider struct {
	nodes []types.ProxyNode
}

func (m *mockNodeProvider) GetAllNodes() []types.ProxyNode {
	return m.nodes
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)

	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}

	checker := NewChecker(&mockRunner{}, cfg)
	np := &mockNodeProvider{
		nodes: []types.ProxyNode{
			{
				Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
			},
		},
	}

	svc := NewService(checker, np, ProberConfig{
		ResourcesPath: "testdata/resources.yaml",
		DBPath:        cfgDir + "/test.db",
	})
	return svc
}

func TestService_RunAll_NoNodes(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, _ := config.Init()
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		ResourcesPath: "testdata/resources.yaml",
		DBPath:        cfgDir + "/test.db",
	})

	err := svc.RunAll(context.Background())
	if err == nil {
		t.Fatal("expected error for no nodes")
	}
}

func TestService_RunForTag_NotFound(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, _ := config.Init()
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		ResourcesPath: "testdata/resources.yaml",
		DBPath:        cfgDir + "/test.db",
	})

	err := svc.RunForTag(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
}

func TestService_GetStatus_Initial(t *testing.T) {
	cfgDir := t.TempDir()
	cfg, _ := config.Init()
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: cfgDir + "/test.db",
	})

	status := svc.GetStatus()
	if status.Status != "idle" {
		t.Errorf("expected idle, got %s", status.Status)
	}
}

func TestService_Stop_Idempotent(t *testing.T) {
	cfgDir := t.TempDir()
	cfg, _ := config.Init()
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: cfgDir + "/test.db",
	})

	svc.Stop() // should not panic
	svc.Stop() // should not panic
}
```

- [ ] **Step 3: Create testdata directory with test YAML**

```yaml
resources:
  - name: youtube
    url: https://www.youtube.com
    type: http
  - name: telegram
    url: https://telegram.org
    type: http
```

Save to `server/internal/resourcecheck/testdata/resources.yaml`.

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/resourcecheck/...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/resourcecheck/service.go server/internal/resourcecheck/service_test.go server/internal/resourcecheck/testdata/
git commit -m "feat(resourcecheck): add service layer with orchestration and scheduler"
```

---

### Task 8: Create resourcecheck — HTTP handlers

**Files:**
- Create: `server/internal/resourcecheck/handler.go`
- Create: `server/internal/resourcecheck/register.go`
- Create: `server/internal/resourcecheck/handler_test.go`

- [ ] **Step 1: Create `handler.go`**

```go
package resourcecheck

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler serves HTTP endpoints for resource checking.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetResources lists configured resources.
func (h *Handler) GetResources(c *gin.Context) {
	resources := h.svc.GetResources()
	c.JSON(http.StatusOK, gin.H{
		"count":     len(resources),
		"resources": resources,
	})
}

// GetResults returns all latest check results.
func (h *Handler) GetResults(c *gin.Context) {
	results, err := h.svc.GetLatestResults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	c.JSON(http.StatusOK, gin.H{
		"count":   len(results),
		"results": results,
	})
}

// GetResultsForTag returns latest results for a specific node tag.
func (h *Handler) GetResultsForTag(c *gin.Context) {
	tag := c.Param("tag")
	results, err := h.svc.GetResultsForTag(tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	c.JSON(http.StatusOK, gin.H{
		"tag":     tag,
		"count":   len(results),
		"results": results,
	})
}

// GetHistory returns check history for a resource + tag combination.
func (h *Handler) GetHistory(c *gin.Context) {
	resource := c.Param("resource")
	tag := c.Param("tag")
	results, err := h.svc.GetHistory(resource, tag, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	c.JSON(http.StatusOK, gin.H{
		"resource": resource,
		"tag":      tag,
		"count":    len(results),
		"results":  results,
	})
}

// Run starts a check operation.
// Body can be: {} (all nodes), {"tag":"..."} (specific node), {"subscription_id":"..."}
func (h *Handler) Run(c *gin.Context) {
	status := h.svc.GetStatus()
	if status.Running {
		c.JSON(http.StatusConflict, gin.H{"error": "check already running"})
		return
	}

	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = RunRequest{} // empty = run all
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.svc.statusMu.Lock()
	h.svc.cancel = cancel
	h.svc.statusMu.Unlock()

	var runErr error
	if req.Tag != "" {
		runErr = h.svc.RunForTag(ctx, req.Tag)
	} else {
		runErr = h.svc.RunAll(ctx)
	}

	if runErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": runErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "check completed"})
}

// Stop cancels a running check.
func (h *Handler) Stop(c *gin.Context) {
	h.svc.Stop()
	c.JSON(http.StatusOK, gin.H{"message": "stop requested"})
}

// Schedule sets up periodic background checking.
func (h *Handler) Schedule(c *gin.Context) {
	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IntervalSec <= 0 {
		h.svc.StopScheduler()
		c.JSON(http.StatusOK, gin.H{"message": "scheduler stopped"})
		return
	}

	h.svc.StartScheduler(req.IntervalSec)
	c.JSON(http.StatusOK, gin.H{
		"message":      "scheduler started",
		"interval_sec": req.IntervalSec,
	})
}

// GetStatus returns the current service status.
func (h *Handler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.GetStatus())
}

// Reload reloads the resources configuration file.
func (h *Handler) Reload(c *gin.Context) {
	if err := h.svc.ReloadResources(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resources reloaded"})
}
```

- [ ] **Step 2: Create `register.go`**

```go
package resourcecheck

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all resource check endpoints.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/resources", h.GetResources)
	rg.GET("/results", h.GetResults)
	rg.GET("/results/:tag", h.GetResultsForTag)
	rg.GET("/history/:resource/:tag", h.GetHistory)
	rg.POST("/run", h.Run)
	rg.POST("/stop", h.Stop)
	rg.POST("/schedule", h.Schedule)
	rg.GET("/status", h.GetStatus)
	rg.POST("/reload", h.Reload)
}
```

- [ ] **Step 3: Create `handler_test.go`**

```go
package resourcecheck

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}

	checker := NewChecker(&mockRunner{}, cfg)
	np := &mockNodeProvider{
		nodes: []types.ProxyNode{
			{
				Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
			},
		},
	}

	svc := NewService(checker, np, ProberConfig{
		DBPath: cfgDir + "/test.db",
	})
	return NewHandler(svc)
}

func TestHandler_GetResources_Empty(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/resourcecheck/resources", h.GetResources)

	req := httptest.NewRequest("GET", "/api/resourcecheck/resources", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp["count"].(float64) != 0 {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestHandler_GetStatus_Idle(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/resourcecheck/status", h.GetStatus)

	req := httptest.NewRequest("GET", "/api/resourcecheck/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var status CheckStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if status.Running {
		t.Error("expected not running")
	}
}

func TestHandler_Run_NoNodes(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, _ := config.Init()
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: cfgDir + "/test.db",
	})
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/resourcecheck/run", h.Run)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/resourcecheck/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Stop_Idempotent(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/resourcecheck/stop", h.Stop)

	req := httptest.NewRequest("POST", "/api/resourcecheck/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_Schedule(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/resourcecheck/schedule", h.Schedule)

	// Start scheduler
	body := `{"interval_sec": 3600}`
	req := httptest.NewRequest("POST", "/api/resourcecheck/schedule", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Stop scheduler
	body = `{"interval_sec": 0}`
	req = httptest.NewRequest("POST", "/api/resourcecheck/schedule", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
```

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/resourcecheck/...`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/resourcecheck/handler.go server/internal/resourcecheck/register.go server/internal/resourcecheck/handler_test.go
git commit -m "feat(resourcecheck): add HTTP handlers and route registration"
```

---

### Task 9: Wire resourcecheck into main.go

**Files:**
- Modify: `server/main.go`

- [ ] **Step 1: Update `main.go`** — add imports, create service, register routes

In `main.go`:

Add imports (in the import block):
```go
"singbox-config-service/internal/resourcecheck"
```

After creating `speedtestSvc` (around line 89), add:
```go
// Create resource checker service
rcChecker := resourcecheck.NewChecker(tr, cfg)
rcSvc := resourcecheck.NewService(rcChecker, subSvc, resourcecheck.ProberConfig{
	ResourcesPath: filepath.Join(cfg.GetDataDir(), "resources.yaml"),
	DBPath:        filepath.Join(cfg.GetDataDir(), "resource_checks.db"),
})
if err := rcSvc.Init(); err != nil {
	log.Printf("Warning: failed to initialize resource checker: %v", err)
}
```

After creating `speedtestHandler` (around line 101), add:
```go
rcHandler := resourcecheck.NewHandler(rcSvc)
```

In the route registration section (around line 152), add:
```go
// Resource check routes
rcGroup := api.Group("/resourcecheck")
rcHandler.RegisterRoutes(rcGroup)
```

Also add `"path/filepath"` to the import block if not already present.

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: success

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: all PASS

- [ ] **Step 4: Run linter**

```bash
golangci-lint run ./...
```

If `golangci-lint` not installed, install:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add server/main.go
git commit -m "feat: wire resource checker service into main application"
```

---

## Spec Coverage Check

| Spec Requirement | Task |
|---|---|
| tunnelrunner: extract from speedtest | Task 1, 2, 3 |
| SQLite store for results | Task 4 (deps), Task 5 (store), Task 9 (init) |
| YAML config for resources | Task 5 (models.LoadResources), Task 7 (service.Init) |
| HTTP resource checking | Task 6 (checker.checkHTTP) |
| TCP resource checking | Task 6 (checker.checkTCP) |
| Per-node + per-resource results | Task 5 (store schema), Task 7 (service) |
| API endpoints (9 routes) | Task 8 (handler + register) |
| Background scheduler | Task 7 (service.StartScheduler) |
| Manual run (all/tag) | Task 8 (handler.Run) |
| Reload resources | Task 8 (handler.Reload) |
| Wiring into main.go | Task 9 |
