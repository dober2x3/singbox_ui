# Backend Vertical Slices Architecture Design

**Date:** 2026-06-01
**Author:** AI Agent
**Status:** Draft

## 1. Executive Summary

Migrate the Go backend from a layered architecture (`handlers/` → `services/` — God package with global singletons) to a **vertical slice architecture** where each domain is an isolated package with its own handler, service, models, and interfaces. Cross-domain communication happens exclusively through interfaces (Dependency Inversion). Shared infrastructure lives in `internal/pkg/`.

## 2. Current Architecture

```
server/
├── main.go              # Entry point, route registration
├── init.go              # Global init: paths, docker, prober
├── handlers/            # 7 files (~1100 LOC) — HTTP handlers
└── services/            # 13 files (~5000 LOC) — God package
```

**Problems:**
- `services/` is a God package with no boundaries
- Global singletons (`dockerService`, `globalProber`) — hidden dependencies, untestable
- Cross-cutting dependencies: `scheduler.go` directly calls `LoadSubscriptions()`, `SaveConfig()`, `ListNamedConfigs()`
- No isolation — impossible to test a single domain without the entire package
- Handlers directly call service functions — no explicit interface boundaries

## 3. Target Architecture

```
server/
├── main.go                   # Entry point, manual DI, route registration
├── internal/
│   ├── pkg/                  # Shared infrastructure (no domain logic)
│   │   ├── docker/           # Docker client wrapper
│   │   ├── config/           # Paths, env vars
│   │   └── types/            # ProxyNode, ProbeNode, SanitizeTag, constants
│   ├── singbox/              # Container lifecycle + config management
│   │   ├── handler.go        # HTTP handlers
│   │   ├── service.go        # Business logic
│   │   ├── models.go         # NamedConfigInfo, etc.
│   │   ├── interfaces.go     # ContainerManager (for scheduler)
│   │   └── register.go       # RegisterRoutes()
│   ├── subscription/         # Subscription CRUD, protocol parsing
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── parser_vmess.go
│   │   ├── parser_clash.go
│   │   ├── models.go
│   │   ├── store.go          # File-based persistence
│   │   └── interfaces.go     # SubscriptionUpdater, NodeProvider
│   ├── prober/               # Node health probing engine
│   │   ├── handler.go
│   │   ├── engine.go         # Prober struct, probe loop
│   │   ├── service.go
│   │   ├── models.go
│   │   └── interfaces.go     # ProbeResultSaver
│   ├── speedtest/            # Proxy speed testing
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── models.go
│   ├── certificate/          # TLS certificates (self-signed, manual)
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── models.go
│   ├── wireguard/            # WireGuard keys + client configs
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── models.go
│   ├── warp/                 # Cloudflare WARP
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── scanner.go        # Endpoint scanner
│   │   └── models.go
│   └── scheduler/            # Subscription auto-update scheduler
│       ├── service.go
│       └── interfaces.go     # SubscriptionUpdater, ContainerManager
└── [handlers/ и services/ удаляются после миграции]
```

## 4. Shared Package (`internal/pkg/`)

### `internal/pkg/docker/`
- `NewClient() (*Client, error)` — creates Docker client from environment
- `Client` wraps Docker SDK operations: ContainerCreate, Start, Stop, Remove, Logs, ImagePull, ImageLoad
- No sing-box specific logic (no config paths, no container naming)
- Provides `ContainerAPI` interface for testability

### `internal/pkg/config/`
- `Init()` — reads `DATA_DIR`, `HOST_DATA_DIR`, `LISTEN_ADDR` env vars
- `GetSingboxDir()`, `GetDataDir()` — path accessors
- `ResolveHostConfigDir(containerPath) (hostPath, error)` — Docker-in-Docker path resolution
- State: single struct, no global vars

### `internal/pkg/types/`
- `ProxyNode` — shared node representation from subscriptions
- `ProbeNode`, `ProbeResult`, `ProbeResultUpdate` — shared probe types
- `SpeedTestResult`, `SpeedTestUpdate` — shared speed test types
- `SanitizeTag(protocol, address, port) string` — tag generation utility
- `PredefinedUserAgents` — User-Agent presets
- `proxyOutboundTypes` — known proxy outbound type whitelist
- `blockedSubscriptionPrefixes` — blocked IP ranges for subscription URL validation

## 5. Domain Slices — Detailed Design

### 5.1 `internal/singbox/`
**Responsibility:** Manage sing-box container lifecycle and configuration files.

**Handler methods:**
- `GET /config` — read config.json
- `POST /config` — save config.json
- `POST /run` — start container
- `POST /stop` — stop container
- `GET /logs` — stream container logs
- `GET /status` — check container running status
- `GET /version` — get sing-box version
- `POST /ensure-image` — ensure Docker image exists
- `GET /instances` — list named configs
- `POST /instances/:name/config` — save named config
- `GET /instances/:name/config` — load named config
- `POST /instances/:name/check` — validate named config
- `DELETE /instances/:name` — delete named config + container
- `POST /instances/:name/run` — start named container
- `POST /instances/:name/stop` — stop named container
- `GET /instances/:name/status` — named container status
- `GET /instances/:name/logs` — named container logs
- `GET /containers` — list all sing-box containers

**Interface:**
```go
type ContainerManager interface {
    UpdateAndRestart(name string, configData []byte) error
    RestartNamed(name string) error
    Status(name string) (running bool, containerID string)
}
```

**Dependencies:** `pkg/docker`, `pkg/config`

### 5.2 `internal/subscription/`
**Responsibility:** Manage proxy subscriptions — CRUD, fetch, parse.

**Handler methods:**
- `GET /` — list all subscriptions
- `POST /` — add subscription
- `POST /:id/refresh` — refresh single subscription
- `PATCH /:id/settings` — update auto-update settings
- `DELETE /:id` — delete subscription
- `POST /refresh-all` — refresh all subscriptions
- `GET /nodes` — get all nodes across subscriptions
- `GET /user-agents` — get predefined UA list

**Parsers:** VMess, VLESS, Trojan, Shadowsocks URL formats + Clash YAML

**Interface:**
```go
type SubscriptionUpdater interface {
    LoadAll() ([]SubscriptionEntry, error)
    UpdateOne(id string) (*SubscriptionEntry, error)
}

type NodeProvider interface {
    GetAllNodes() ([]ProxyNode, error)
}

type ProbeResultSaver interface {
    SaveProbeResults(results []ProbeResultUpdate) error
}

type SpeedTestResultSaver interface {
    SaveSpeedTestResults(results []SpeedTestUpdate) error
}
```

**Dependencies:** `pkg/types`, `pkg/config`

### 5.3 `internal/prober/`
**Responsibility:** Async node health probing with TCP/HTTP probes, sliding window success rate.

**Handler methods:**
- `GET /status` — prober stats
- `GET /results` — all probe results
- `GET /results/:tag` — single node result
- `GET /best` — best (lowest latency) node
- `GET /online` — all online nodes
- `POST /nodes` — add probe node
- `PUT /nodes` — batch update nodes
- `DELETE /nodes/:tag` — remove node
- `DELETE /nodes` — clear all nodes
- `POST /start` — start prober
- `POST /stop` — stop prober
- `POST /sync` — sync nodes from subscription
- `POST /save` — save results to subscription

**Engine:** Prober struct with goroutine loop, semaphore concurrency control, ring buffer history.

**Interface:** `ProbeResultSaver` (injected from subscription)

**Dependencies:** `pkg/types`, `pkg/config`

### 5.4 `internal/speedtest/`
**Responsibility:** Serial proxy node speed testing via temporary sing-box containers.

**Handler methods:**
- `POST /start` — start speed test
- `GET /status` — get current status/results
- `POST /stop` — cancel speed test

**Key logic:** `testOneNode()` — creates temp container with SOCKS/HTTP proxy, measures latency and download throughput.

**Dependencies:** `pkg/docker`, `pkg/types`, `pkg/config`

### 5.5 `internal/certificate/`
**Responsibility:** TLS certificate management (self-signed, manual, Reality keys).

**Handler methods:** (shared with singbox route group)
- `POST /certificate` — generate self-signed cert
- `GET /certificate` — get certificate info
- `POST /certificate/upload` — upload cert+key
- `POST /reality/keypair` — generate Reality keypair
- `POST /reality/public-key` — derive Reality public key
- `POST /reality/check-tls` — check TLS 1.3 support

**Dependencies:** `pkg/config`

### 5.6 `internal/wireguard/`
**Responsibility:** WireGuard key generation, IP-bound key cache, client config management.

**Handler methods:**
- `POST /keygen` — generate WireGuard keypair with cache
- `POST /pubkey` — derive public key from private key
- `GET /keys-cache` — list cached keys
- `GET /public-ip` — get server's public IP
- `GET /client-config` — get client config
- `POST /client-config` — save client config
- `POST /save-client-file` — save .conf file
- `GET /client-files` — list .conf files

**Dependencies:** `pkg/config`

### 5.7 `internal/warp/`
**Responsibility:** Cloudflare WARP device registration, WARP+ license binding, endpoint scanning.

**Handler methods:**
- `GET /account` — get WARP account info
- `DELETE /account` — delete local WARP record
- `POST /register` — register WARP device
- `POST /license` — bind WARP+ license
- `POST /scan` — scan optimal WARP endpoints

**Scanner:** Real WireGuard handshake probe across 8 Cloudflare /24 subnets × 54 ports.

**Dependencies:** `pkg/config`

### 5.8 `internal/scheduler/`
**Responsibility:** Background goroutine that checks subscription auto-update intervals and refreshes + applies to running containers.

**Interface dependencies:**
```go
type SubscriptionUpdater interface {
    LoadAll() ([]SubscriptionEntry, error)
    UpdateOne(id string) (*SubscriptionEntry, error)
}

type ContainerManager interface {
    UpdateAndRestart(name string, configData []byte) error
    Status(name string) (running bool, containerID string)
}
```

**No HTTP handlers.** Started as goroutine in `main.go`.

**Dependencies:** Interfaces only (no direct domain package imports)

## 6. Dependency Injection (main.go)

```go
func main() {
    // 1. Init shared
    cfg := config.Init()
    dockerClient, _ := docker.NewClient()

    // 2. Create domain services
    singboxSvc := singbox.NewService(dockerClient, cfg)
    subscriptionSvc := subscription.NewService(cfg)
    proberSvc := prober.NewService(cfg)
    speedtestSvc := speedtest.NewService(dockerClient, cfg)
    certificateSvc := certificate.NewService(cfg)
    wireguardSvc := wireguard.NewService(cfg)
    warpSvc := warp.NewService(cfg)

    // 3. Start background workers
    sched := scheduler.New(singboxSvc, subscriptionSvc)
    sched.Start()

    // 4. Create handlers with interface wiring
    singboxHandler := singbox.NewHandler(singboxSvc)
    subscriptionHandler := subscription.NewHandler(subscriptionSvc)
    proberHandler := prober.NewHandler(proberSvc, subscriptionSvc) // subscriptionSvc implements ProbeResultSaver
    speedtestHandler := speedtest.NewHandler(speedtestSvc, subscriptionSvc)
    certificateHandler := certificate.NewHandler(certificateSvc)
    wireguardHandler := wireguard.NewHandler(wireguardSvc)
    warpHandler := warp.NewHandler(warpSvc)

    // 5. Route registration
    r := gin.Default()
    api := r.Group("/api")

    singboxHandler.RegisterRoutes(api.Group("/singbox"))
    subscriptionHandler.RegisterRoutes(api.Group("/subscription"))
    proberHandler.RegisterRoutes(api.Group("/prober"))
    speedtestHandler.RegisterRoutes(api.Group("/speedtest"))
    certificateHandler.RegisterRoutes(api.Group("/singbox"))
    wireguardHandler.RegisterRoutes(api.Group("/wireguard"))
    warpHandler.RegisterRoutes(api.Group("/warp"))

    // Health check
    r.GET("/health", healthHandler)

    // Static files, CORS, listen...
}
```

## 7. Interface Contracts (Cross-Domain)

| Interface | Defined In | Implemented By | Used By |
|-----------|-----------|----------------|---------|
| `ContainerManager` | `internal/singbox/interfaces.go` | `singbox.Service` | `scheduler` |
| `SubscriptionUpdater` | `internal/subscription/interfaces.go` | `subscription.Service` | `scheduler` |
| `NodeProvider` | `internal/subscription/interfaces.go` | `subscription.Service` | `prober`, `speedtest` |
| `ProbeResultSaver` | `internal/subscription/interfaces.go` | `subscription.Service` | `prober` |
| `SpeedTestResultSaver` | `internal/subscription/interfaces.go` | `subscription.Service` | `speedtest` |
| `ContainerAPI` | `internal/pkg/docker/interfaces.go` | `docker.Client` | `singbox`, `speedtest` |

## 8. Migration Plan

| Step | Action | Tests Written | Verification |
|------|--------|-------------|-------------|
| 1 | Create `internal/pkg/{docker,config,types}` — extract shared code | `pkg/types/*_test.go`, `pkg/config/*_test.go` | `go build ./...` + `go test ./internal/pkg/...` |
| 2 | Create `internal/{wireguard,certificate,warp}` — independent slices | Full coverage of all public methods in each package | `go build ./...` + `go test ./internal/...` |
| 3 | Create `internal/subscription/` — extract parsing + CRUD | `parser_*_test.go`, `service_test.go`, `store_test.go` | `go build ./...` + `go test ./internal/subscription/...` |
| 4 | Create `internal/singbox/` — extract container management | `service_test.go` (with docker mock) | `go build ./...` + `go test ./internal/singbox/...` |
| 5 | Create `internal/{prober,speedtest}/` — extract probing + speed test | `engine_test.go`, `service_test.go` (adapt existing) | `go build ./...` + `go test ./internal/prober/... ./internal/speedtest/...` |
| 6 | Create `internal/scheduler/` — extract auto-update scheduler | `service_test.go` (full mock of both dependencies) | `go build ./...` + `go test ./internal/scheduler/...` |
| 7 | Rewrite `main.go` — new DI, route registration | — | `go run .` (manual smoke test) |
| 8 | Delete `handlers/` and `services/` directories | — | `go build ./...` |
| 9 | Run all tests + race detector | — | `go test -race -cover ./...` |
| 10 | Run linter | — | `golangci-lint run ./...` |
| 11 | Build full binary + frontend | — | `npm run build` + `go build -o sing-box-ui .` |

**Rollback strategy:** Each step compiles independently. Old `handlers/` and `services/` are only deleted on step 8. If something breaks, step 7's `main.go` can fall back to the old packages temporarily.

## 9. File Size Estimates

| Slice | Current (LOC) | Estimated New (LOC) |
|-------|--------------|-------------------|
| `pkg/docker` | ~780 (docker.go) | ~300 (extracted interface + core) |
| `pkg/config` | ~30 (init.go paths) + ~40 (scattered) | ~80 |
| `pkg/types` | ~30 types scattered | ~150 |
| `singbox` | ~480 (singbox.go) + ~380 (handlers) | ~500 |
| `subscription` | ~1440 (subscription.go) + ~210 (handlers) | ~900 |
| `prober` | ~650 (prober.go) + ~400 (handlers) | ~700 |
| `speedtest` | ~380 + ~200 (handlers) | ~400 |
| `certificate` | ~190 + ~100 (handlers) | ~200 |
| `wireguard` | ~370 + ~150 (handlers) | ~350 |
| `warp` + scanner | ~730 + ~200 (handlers) | ~600 |
| `scheduler` | ~300 | ~200 |
| `main.go` | ~200 | ~150 |
| **Total** | **~5500** | **~4500** |

## 10. Testing Strategy — 100% Coverage of All Public Methods

### 10.1 Mandate

Every exported function, method, type, and interface in every package MUST have corresponding tests. No public API is considered complete without tests.

Exceptions (must be explicitly documented in code):
- Methods that require real Docker daemon (marked `// TestRequiresDocker`)
- Methods that require real network access (marked `// TestRequiresNetwork`)
- These exceptions still need a unit test with mocked dependencies that validates logic

### 10.2 Test Layers

#### Layer 1: Pure Logic Tests (no mocks needed)

Applies to: `pkg/types`, `internal/subscription/parser_*`, `internal/wireguard/`, `internal/certificate/`

These are deterministic, side-effect-free functions. Every public function gets:
- **Happy path** — normal input → expected output
- **Edge cases** — empty strings, zero values, extreme values
- **Error cases** — invalid input → expected error
- **Special formats** — IPv4, IPv6, URL-encoded, base64 variants

```go
// Example: pkg/types/sanitize_tag_test.go
func TestSanitizeTag_ipv4(t *testing.T)
func TestSanitizeTag_ipv6(t *testing.T)
func TestSanitizeTag_specialChars(t *testing.T)
func TestSanitizeTag_emptyAddress(t *testing.T)

// Example: subscription/parser_vmess_test.go
func TestParseVMessNode_standard(t *testing.T)
func TestParseVMessNode_withTLS(t *testing.T)
func TestParseVMessNode_withWSTransport(t *testing.T)
func TestParseVMessNode_invalidBase64(t *testing.T)
func TestParseVMessNode_missingFields(t *testing.T)
```

#### Layer 2: Service Logic Tests (mocked dependencies)

Applies to: `internal/subscription/service.go`, `internal/prober/service.go`, `internal/speedtest/service.go`,
`internal/singbox/service.go`, `internal/scheduler/service.go`, `internal/warp/service.go`

Each service's public methods are tested with every dependency mocked via the interface contracts:

```go
// For each public method:
func TestSubscriptionService_Add(t *testing.T)    // mock store
func TestSubscriptionService_Add_invalidURL(t *testing.T)
func TestSubscriptionService_Refresh(t *testing.T)  // mock HTTP fetcher
func TestSubscriptionService_Delete_notFound(t *testing.T)

func TestProberService_AddNode(t *testing.T)
func TestProberService_GetResults_empty(t *testing.T)
func TestProberService_SaveResults(t *testing.T)   // mock ProbeResultSaver
```

#### Layer 3: Engine/Stateful Tests

Applies to: `internal/prober/engine.go`, `internal/warp/scanner.go`

Tests for goroutine lifecycle, concurrency, context cancellation:

```go
func TestProberEngine_StartStop(t *testing.T)
func TestProberEngine_ContextCancellation(t *testing.T)
func TestProberEngine_ConcurrentAccess(t *testing.T)
func TestWarpHandshakeProbe_success(t *testing.T)
func TestWarpHandshakeProbe_timeout(t *testing.T)
```

#### Layer 4: Handler Tests (Gin test framework)

Applies to: `internal/*/handler.go`

Each handler is tested with `httptest.NewRecorder()` + `gin.CreateTestContext()`:

```go
func TestSingboxHandler_GetConfig(t *testing.T)
func TestSubscriptionHandler_Add(t *testing.T)
func TestProberHandler_GetStatus(t *testing.T)
```

### 10.3 Mock Strategy

| Dependency | How to mock |
|------------|-------------|
| Docker client (`ContainerAPI`) | Generate mock via `go:generate` or hand-written in `internal/pkg/docker/mock_test.go` |
| File system | In-memory implementations of store interfaces (e.g. `memStore` for subscription) |
| HTTP fetcher | `SubscriptionFetcher` interface — mock returns canned data |
| Time | Use `clock.Clock` interface or accept `time.Now` as parameter in testable code |
| Prober network probes | Replace `tcpProbe`/`httpProbe` methods with overridable function fields in tests |

### 10.4 Coverage Target

| Package | Target | Method |
|---------|--------|--------|
| `pkg/types` | 100% | Table-driven tests for all exported functions |
| `pkg/config` | 100% | Tests with env var setup/teardown |
| `pkg/docker` | 100% | Interface mock; real client only if Docker present |
| `internal/subscription/` | 100% | All parsers + service methods |
| `internal/prober/` | 100% | Engine + service + CRUD |
| `internal/singbox/` | 100% | Service with docker mock |
| `internal/speedtest/` | 100% | Service with docker mock |
| `internal/certificate/` | 100% | Pure crypto tests |
| `internal/wireguard/` | 100% | Key generation + cache tests |
| `internal/warp/` | 100% | Registration mock + scanner |
| `internal/scheduler/` | 100% | Full mock of both dependencies |
| **Total** | **100% of public API** | |

### 10.5 Tooling

- `go test -coverprofile=coverage.out ./...` — run all tests
- `go tool cover -html=coverage.out` — visual coverage report
- Table-driven tests (`t.Run()`) for all pure logic
- Test helpers in `internal/testutil/` for common mocks
- Race detector: `go test -race ./...` must pass

### 10.6 Migration — Keeping Old Tests Green

During migration, old tests continue to exist until step 8 (when `services/` is deleted). New tests are written alongside new code. The final state:
- Old tests deleted with `services/` and `handlers/`
- New tests cover all public methods in all new packages
- CI pipeline runs `go test -cover ./...` and enforces coverage

## 11. Key Decisions

1. **No global singletons** — all dependencies are explicit, passed via constructors
2. **Interface definitions live in the consuming package** — `scheduler` defines what it needs, not the provider
3. **`init.go` removed** — its logic distributed: paths → `config.Init()`, docker → `docker.NewClient()`, prober → `prober.NewService()`, scheduler → `scheduler.Start()`
4. **Each handler struct holds a reference to its service** — no more `services.GetProber()` calls from handlers
5. **One `register.go` per slice** — each domain registers its own routes, keeping `main.go` clean

## 12. Future Considerations

- **Graceful shutdown:** `scheduler.Stop()` + `dockerClient.Close()` on SIGTERM
- **Metrics:** Each slice can expose its own metrics if needed
- **Event bus:** If cross-domain communication grows, replace direct interface calls with an event bus
