# Optional Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--dashboard` CLI flag to make frontend serving optional, default disabled.

**Architecture:** `main.go` parses `--dashboard` flag with Go's `flag` package. If passed, `setupStaticFiles()` registers the `NoRoute` handler. Config package unchanged. Docker Compose uses `command:` to pass the flag when needed.

**Tech Stack:** Go 1.24, Gin, Docker Compose

---

### Task 1: Add `--dashboard` flag to main.go

**Files:**
- Modify: `server/main.go`

- [ ] **Step 1: Add `flag` import**

```go
import (
	"context"
	"embed"
	"flag"     // <-- add
	"io/fs"
	"log"
	"net/http"
```

- [ ] **Step 2: Parse `--dashboard` flag at the start of `main()`**

```go
func main() {
	// Parse CLI flags
	serveDashboard := flag.Bool("dashboard", false, "Serve embedded frontend dashboard")
	flag.Parse()
```

- [ ] **Step 3: Replace unconditional `setupStaticFiles(r)` with conditional**

```go
	// Static file server (optional, controlled by --dashboard flag)
	if *serveDashboard {
		setupStaticFiles(r)
		log.Println("Dashboard static files are being served")
	} else {
		log.Println("Dashboard serving disabled (use --dashboard to enable)")
	}
```

- [ ] **Step 4: Build and smoke test**

Run: `cd server && go build -o /tmp/singbox-test .`

```bash
# Without flag — dashboard disabled
/tmp/singbox-test &
curl http://127.0.0.1:7000/health     # → 200
curl -o /dev/null -w "%{http_code}" http://127.0.0.1:7000/  # → 404
kill %1

# With flag — dashboard enabled
/tmp/singbox-test --dashboard &
curl http://127.0.0.1:7000/health     # → 200
curl -o /dev/null -w "%{http_code}" http://127.0.0.1:7000/  # → 200
kill %1
```

Expected: both scenarios work correctly.

---

### Task 2: Revert env var changes from config

**Files:**
- Modify: `server/internal/pkg/config/config.go`
- Modify: `server/internal/pkg/config/config_test.go`

- [ ] **Step 1: Remove `dashboardEnabled` field from Config struct**

```go
type Config struct {
	dataDir     string
	hostDataDir string
	listenAddr  string
	singboxDir  string
	// dashboardEnabled bool  <-- remove
}
```

- [ ] **Step 2: Remove `IsDashboardEnabled` getter**

Remove the `IsDashboardEnabled()` method from config.go.

- [ ] **Step 3: Remove `DASHBOARD_ENABLED` env var parsing from `Init()`**

Remove the `dashboardEnabled` variable block and the field from the struct literal.

- [ ] **Step 4: Remove `TestIsDashboardEnabled_*` tests from config_test.go**

Remove all 4 `TestIsDashboardEnabled_*` test functions.

- [ ] **Step 5: Run tests**

Run: `cd server && go test ./internal/pkg/config/ -v`
Expected: original 7 tests pass.

---

### Task 3: Update docker-compose.yml

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Replace env var comments with `command:` approach**

```yaml
    environment:
      - DATA_DIR=/home/data
      - HOST_DATA_DIR=${PWD}/data
      - LISTEN_ADDR=0.0.0.0:7000
      - TZ=Asia/Shanghai
    # Default: backend only (TUI connects from another machine).
    # For frontend+backend on same machine, uncomment:
    # command: ["./sing-box-ui", "--dashboard"]
```

---

### Task 4: Final verification

- [ ] **Step 1: Run full suite**

```bash
cd server
PATH="$HOME/go/bin:$PATH" golangci-lint run ./...   # no errors
go build ./...                                        # no errors
go test ./...                                         # all pass
```

- [ ] **Step 2: Push branch**

```bash
git push origin feature/optional-dashboard
```
