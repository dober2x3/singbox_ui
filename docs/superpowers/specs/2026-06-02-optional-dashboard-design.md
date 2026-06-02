# Optional Dashboard & External API Access

**Date:** 2026-06-02
**Status:** Specified

## Problem

The sing-box UI currently bundles frontend and backend into a single binary. The frontend
is always served, and the HTTP server binds only to `127.0.0.1`. Two deployment scenarios
require changes:

1. **Backend-only deployment** — run the backend without the web dashboard (a TUI client
   connects via the API from another machine).
2. **Mixed deployment** — frontend + backend on one machine, but the backend must also be
   reachable from other machines on the network (for the TUI client).

## Scope

This spec covers changes to the Go backend and Docker Compose configuration only. The
frontend codebase is untouched. The TUI client is a separate future project.

---

## 1. `--dashboard` CLI Flag

### Behaviour

A new command-line flag controls whether the embedded dashboard (static frontend files)
is served at runtime.

| Flag | Default | Description |
|------|---------|-------------|
| `--dashboard` | `false` (not set) | When passed, the `NoRoute` static file handler is registered. API endpoints are always functional regardless. |

The frontend remains embedded in the Go binary at compile time (via `//go:embed`). This
keeps build and CI unchanged. The ~1.6 MB overhead of the embedded files in a
backend-only deployment is considered acceptable.

### Files Changed

**`server/main.go`**

- Add `flag` import.
- Parse `--dashboard` flag at the start of `main()`.
- Replace unconditional `setupStaticFiles(r)` with a conditional check.

```go
serveDashboard := flag.Bool("dashboard", false, "Serve embedded frontend dashboard")
flag.Parse()

// ... later ...

if *serveDashboard {
    setupStaticFiles(r)
    log.Println("Dashboard static files are being served")
} else {
    log.Println("Dashboard serving disabled (use --dashboard to enable)")
}
```

**No changes to** `server/internal/pkg/config/config.go` or its tests.

---

## 2. External API Access

The `LISTEN_ADDR` environment variable already fully supports binding to any address.
No code changes are required. The default remains `127.0.0.1:7000` for security; users
set `LISTEN_ADDR=0.0.0.0:7000` to enable external access.

The existing CORS middleware (`Access-Control-Allow-Origin: *`) is already correct for
all scenarios (browser and non-browser clients). No changes.

---

## 3. Docker Compose

**File:** `docker-compose.yml`

### Scenario A — Backend only (TUI on another machine)

```yaml
services:
  singbox-ui:
    image: ghcr.io/spadesa99/singbox_ui:latest
    container_name: singbox-ui
    restart: unless-stopped
    network_mode: host
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/home/data
    environment:
      - DATA_DIR=/home/data
      - HOST_DATA_DIR=${PWD}/data
      - LISTEN_ADDR=0.0.0.0:7000
      - TZ=Asia/Shanghai
    # command not overridden → --dashboard not passed → backend only
```

### Scenario B — Frontend + Backend (same machine)

```yaml
services:
  singbox-ui:
    image: ghcr.io/spadesa99/singbox_ui:latest
    container_name: singbox-ui
    restart: unless-stopped
    network_mode: host
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/home/data
    environment:
      - DATA_DIR=/home/data
      - HOST_DATA_DIR=${PWD}/data
      - LISTEN_ADDR=127.0.0.1:7000
      - TZ=Asia/Shanghai
    command: ["./sing-box-ui", "--dashboard"]
```

---

## 4. No Changes

- **Config struct** — no changes (reverted from env var approach).
- **Config tests** — no changes (reverted).
- **CORS** — already permissive (`*`), suitable for all scenarios.
- **API handlers** — untouched.
- **Embed directive** (`//go:embed dist/*`) — untouched.
- **Dockerfile / CI / deploy.sh** — untouched.
- **Frontend code** — untouched.

---

## 5. Verification

1. `./sing-box-ui` — API responds, dashboard returns 404.
2. `./sing-box-ui --dashboard` — API responds, dashboard is served.
3. `LISTEN_ADDR=0.0.0.0:7000` — server responds from another host on the network.
