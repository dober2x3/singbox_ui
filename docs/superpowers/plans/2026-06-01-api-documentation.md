# API Documentation (Swagger/OpenAPI) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add automatic OpenAPI (Swagger) documentation to the Go backend using swaggo annotations, served via Swagger UI at `/swagger/index.html`.

**Architecture:** swaggo reads Go comment annotations (`// @Summary`, `// @Router`, etc.) and generates OpenAPI 2.0 JSON/YAML specs. A gin-swagger middleware serves the Swagger UI. Inline request/response structs in handlers are refactored into named types in `models.go` files with swaggo `example` and `enums` tags.

**Tech Stack:** Go 1.24, Gin, swaggo/swag, gin-swagger, swaggo/files

---

### Task 0: Foundation — install swaggo, create docs package, wire up route

**Files:**
- Modify: `server/go.mod`
- Create: `server/internal/docs/docs.go`
- Create: `server/internal/docs/errors.go`
- Modify: `server/main.go`

- [ ] **Step 0.1: Install swaggo CLI and add dependencies**

```bash
cd /home/kev/work/projects/singbox_ui/server
go install github.com/swaggo/swag/cmd/swag@latest
go get github.com/swaggo/gin-swagger@latest
go get github.com/swaggo/files@latest
```

Expected: `go get` adds dependencies to `go.mod` and `go.sum`.

- [ ] **Step 0.2: Create `server/internal/docs/docs.go` with root swaggo annotations**

```go
// Package docs provides the OpenAPI (Swagger) documentation for the Singbox UI API.
//
// @title           Singbox UI API
// @version         1.0
// @description     API for managing sing-box, WireGuard, subscriptions, probing, and WARP
// @host            localhost:8080
// @BasePath        /api
// @schemes         http
//
// @tag.name        subscription
// @tag.description Subscription management
// @tag.name        singbox
// @tag.description sing-box container lifecycle and config management
// @tag.name        wireguard
// @tag.description WireGuard key generation and client configuration
// @tag.name        certificate
// @tag.description Self-signed certificate management
// @tag.name        reality
// @tag.description REALITY keypair generation and TLS checks
// @tag.name        prober
// @tag.description Node health probing and latency measurement
// @tag.name        speedtest
// @tag.description Proxy speed testing
// @tag.name        warp
// @tag.description WARP registration, licensing, and endpoint scanning
// @tag.name        system
// @tag.description System health and status endpoints
package docs
```

- [ ] **Step 0.3: Create `server/internal/docs/errors.go` with shared error type**

```go
package docs

// ErrorResponse is the standard error response returned by the API.
type ErrorResponse struct {
	Error   string `json:"error" example:"internal server error"`
	Message string `json:"message,omitempty" example:"detailed error description"`
}
```

- [ ] **Step 0.4: Add swagger route and docs import to `server/main.go`**

Add imports (after line ~17):
```go
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "singbox-config-service/internal/docs"
```

Add route (before health check, after line ~140):
```go
	// Swagger API documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

- [ ] **Step 0.5: Run `swag init` to verify generation works**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
```

Expected: generates `internal/docs/swagger.json`, `internal/docs/swagger.yaml`, `internal/docs/docs.go` (auto-generated).

- [ ] **Step 0.6: Verify build compiles**

```bash
cd /home/kev/work/projects/singbox_ui/server
go build ./...
```

- [ ] **Step 0.7: Commit**

```bash
git add server/go.mod server/go.sum server/internal/docs/ server/main.go
git commit -m "feat: add swaggo foundation with Swagger UI route"
```

---

### Task 1: certificate domain (5 endpoints + 3 reality endpoints)

**Files:**
- Modify: `server/internal/certificate/handler.go`
- Modify: `server/internal/certificate/models.go`
- Modify: `server/internal/certificate/reality.go`

- [ ] **Step 1.1: Add swaggo annotations to `models.go`**

```go
package certificate

// CertificateInfo contains metadata about a TLS certificate.
// @Description Certificate metadata including validity period and fingerprint
type CertificateInfo struct {
	CertPath    string `json:"cert_path" example:"/data/singbox/cert.pem"`
	KeyPath     string `json:"key_path" example:"/data/singbox/key.pem"`
	CommonName  string `json:"common_name" example:"example.com"`
	ValidFrom   string `json:"valid_from" example:"2026-06-01T00:00:00Z"`
	ValidTo     string `json:"valid_to" example:"2027-06-01T00:00:00Z"`
	Fingerprint string `json:"fingerprint" example:"SHA256:abc123..."`

// GenerateCertRequest request body for generating a self-signed certificate
// @Description Request to generate a self-signed TLS certificate
type GenerateCertRequest struct {
	Domain    string `json:"domain" example:"example.com" binding:"required"`
	ValidDays int    `json:"valid_days" example:"365" binding:"required"`
}

// RealityKeypairResponse response for Reality keypair generation
// @Description x25519 key pair for Reality TLS
type RealityKeypairResponse struct {
	PrivateKey string `json:"private_key" example:"ARVXKXp6V9XmXQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQ"`
	PublicKey  string `json:"public_key" example:"9XmXQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQ"`
}

// DerivePublicKeyRequest request body for deriving a Reality public key
// @Description Request to derive a public key from a private key
type DerivePublicKeyRequest struct {
	PrivateKey string `json:"private_key" example:"ARVXKXp6V9XmXQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQmQ" binding:"required"`
}

// CheckTLS13Request request body for checking TLS 1.3 support
// @Description Request to check if a server supports TLS 1.3
type CheckTLS13Request struct {
	Server string `json:"server" example:"example.com" binding:"required"`
	Port   int    `json:"port" example:"443"`
}

// CheckTLS13Response response for TLS 1.3 support check
// @Description Result of TLS 1.3 support check
type CheckTLS13Response struct {
	Supported  bool   `json:"supported" example:"true"`
	TLSVersion string `json:"tls_version,omitempty" example:"TLS 1.3"`
	Error      string `json:"error,omitempty" example:"connection failed"`
}
```

- [ ] **Step 1.2: Add annotations to `handler.go`**

`GenerateSelfSignedCert`:
```go
// GenerateSelfSignedCert generates a self-signed TLS certificate.
// @Summary      Generate self-signed certificate
// @Description  Generates a self-signed TLS certificate for the given domain and validity period, saves it to the singbox directory
// @Tags         certificate
// @Accept       json
// @Produce      json
// @Param        request body GenerateCertRequest true "Domain and validity period"
// @Success      200  {object}  CertificateInfo
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/certificate [post]
func (h *Handler) GenerateSelfSignedCert(c *gin.Context) {
```

`GetCertificateInfo`:
```go
// GetCertificateInfo returns information about the currently stored certificate.
// @Summary      Get certificate info
// @Description  Returns metadata about the stored TLS certificate including validity and fingerprint
// @Tags         certificate
// @Produce      json
// @Success      200  {object}  CertificateInfo
// @Failure      404  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/certificate [get]
func (h *Handler) GetCertificateInfo(c *gin.Context) {
```

`UploadCertificate`:
```go
// UploadCertificate handles uploading cert.pem and key.pem files.
// @Summary      Upload certificate files
// @Description  Uploads cert.pem and key.pem files as multipart form data
// @Tags         certificate
// @Accept       mpfd
// @Produce      json
// @Param        cert formData file true "Certificate file (cert.pem)"
// @Param        key formData file true "Private key file (key.pem)"
// @Success      200  {object}  CertificateInfo
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/certificate/upload [post]
func (h *Handler) UploadCertificate(c *gin.Context) {
```

- [ ] **Step 1.3: Add annotations to `reality.go`**

`GenerateRealityKeypair`:
```go
// GenerateRealityKeypair generates a x25519 key pair for Reality TLS
// @Summary      Generate Reality key pair
// @Description  Generates a random x25519 key pair for use with Reality TLS
// @Tags         reality
// @Produce      json
// @Success      200  {object}  RealityKeypairResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/reality/keypair [post]
func GenerateRealityKeypair(c *gin.Context) {
```

`DeriveRealityPublicKey`:
```go
// DeriveRealityPublicKey derives a public key from a Reality private key
// @Summary      Derive Reality public key
// @Description  Derives the public key from a base64-encoded x25519 private key
// @Tags         reality
// @Accept       json
// @Produce      json
// @Param        request body DerivePublicKeyRequest true "Private key"
// @Success      200  {object}  RealityKeypairResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/reality/public-key [post]
func DeriveRealityPublicKey(c *gin.Context) {
```

`CheckTLS13Support`:
```go
// CheckTLS13Support checks if a server supports TLS 1.3
// @Summary      Check TLS 1.3 support
// @Description  Checks if a remote server supports TLS 1.3 (required for Reality disguise domain)
// @Tags         reality
// @Accept       json
// @Produce      json
// @Param        request body CheckTLS13Request true "Server domain and port"
// @Success      200  {object}  CheckTLS13Response
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /singbox/reality/check-tls [post]
func CheckTLS13Support(c *gin.Context) {
```

- [ ] **Step 1.4: Add `docs` import to handler.go and reality.go**

In `handler.go` add import:
```go
	"singbox-config-service/internal/docs"
```

In `reality.go` add import:
```go
	"singbox-config-service/internal/docs"
```

- [ ] **Step 1.5: Generate docs and verify build**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
```

- [ ] **Step 1.6: Commit**

```bash
git add server/internal/certificate/ server/internal/docs/
git commit -m "feat: add swagger annotations for certificate domain"
```

---

### Task 2: speedtest domain (3 endpoints)

**Files:**
- Modify: `server/internal/speedtest/handler.go`
- Create: `server/internal/speedtest/models.go`

- [ ] **Step 2.1: Create `server/internal/speedtest/models.go`**

```go
package speedtest

// docs imports will be added
import "singbox-config-service/internal/docs"

// StartSpeedTestRequest request body for starting a speed test
// @Description Request to start a speed test for a proxy node
type StartSpeedTestRequest struct {
	Tag      string `json:"tag" example:"my-proxy-node" binding:"required"`
	ServerID string `json:"server_id" example:"us-east-1"`
}

// SpeedTestStatus contains the current speed test status
// @Description Current speed test execution status
type SpeedTestStatus struct {
	Running    bool   `json:"running" example:"true"`
	Progress   string `json:"progress,omitempty" example:"3/5 complete"`
	CurrentTag string `json:"current_tag,omitempty" example:"my-proxy-node"`
}
```

- [ ] **Step 2.2: Add annotations to `handler.go`**

Read existing handler, add annotations:
```go
// StartSpeedTest starts a speed test for the given proxy node.
// @Summary      Start speed test
// @Description  Starts a speed test for a specific proxy node by tag
// @Tags         speedtest
// @Accept       json
// @Produce      json
// @Param        request body StartSpeedTestRequest true "Proxy node tag and optional server ID"
// @Success      200  {object}  map[string]interface{}  "speed test started"
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /speedtest/start [post]
```

```go
// GetSpeedTestStatus returns the current speed test status.
// @Summary      Get speed test status
// @Description  Returns whether a speed test is currently running and its progress
// @Tags         speedtest
// @Produce      json
// @Success      200  {object}  SpeedTestStatus
// @Router       /speedtest/status [get]
```

```go
// StopSpeedTest stops the currently running speed test.
// @Summary      Stop speed test
// @Description  Stops the currently running speed test if one is in progress
// @Tags         speedtest
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "speeds test stopped"
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /speedtest/stop [post]
```

- [ ] **Step 2.3: Generate docs and verify build**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
```

- [ ] **Step 2.4: Commit**

```bash
git add server/internal/speedtest/
git commit -m "feat: add swagger annotations for speedtest domain"
```

---

### Task 3: wireguard domain (8 endpoints)

**Files:**
- Modify: `server/internal/wireguard/handler.go`
- Modify: `server/internal/wireguard/models.go`

- [ ] **Step 3.1: Read existing `models.go` and enhance with swaggo tags**

- [ ] **Step 3.2: Read `handler.go` and add swaggo annotations to all 8 endpoints**

- [ ] **Step 3.3: Generate docs and verify build**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
```

- [ ] **Step 3.4: Commit**

```bash
git add server/internal/wireguard/
git commit -m "feat: add swagger annotations for wireguard domain"
```

---

### Task 4: warp domain (5 endpoints)

**Files:**
- Modify: `server/internal/warp/handler.go`
- Modify: `server/internal/warp/models.go`

- [ ] **Step 4.1: Read existing `models.go` and enhance with swaggo tags**

- [ ] **Step 4.2: Read `handler.go` and add swaggo annotations to all 5 endpoints**

- [ ] **Step 4.3: Generate docs and verify build**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
```

- [ ] **Step 4.4: Commit**

```bash
git add server/internal/warp/
git commit -m "feat: add swagger annotations for warp domain"
```

---

### Task 5: subscription domain (7 endpoints)

**Files:**
- Modify: `server/internal/subscription/handler.go`
- Modify: `server/internal/subscription/models.go`

- [ ] **Step 5.1: Read existing `models.go` and enhance with swaggo tags + examples**

- [ ] **Step 5.2: Read `handler.go` and add swaggo annotations to all 7 endpoints**

- [ ] **Step 5.3: Generate docs and verify build**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
```

- [ ] **Step 5.4: Commit**

```bash
git add server/internal/subscription/
git commit -m "feat: add swagger annotations for subscription domain"
```

---

### Task 6: prober domain (12 endpoints)

**Files:**
- Modify: `server/internal/prober/handler.go`
- Modify: `server/internal/prober/models.go`

- [ ] **Step 6.1: Read existing `models.go` and enhance with swaggo tags**

- [ ] **Step 6.2: Read `handler.go` and add swaggo annotations to all 12 endpoints**

- [ ] **Step 6.3: Generate docs and verify build**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
```

- [ ] **Step 6.4: Commit**

```bash
git add server/internal/prober/
git commit -m "feat: add swagger annotations for prober domain"
```

---

### Task 7: singbox domain (15 endpoints)

**Files:**
- Modify: `server/internal/singbox/handler.go`
- Modify: `server/internal/singbox/models.go`

- [ ] **Step 7.1: Read existing `models.go` and enhance with swaggo tags**

- [ ] **Step 7.2: Read `handler.go` and add swaggo annotations to all 15 endpoints**

- [ ] **Step 7.3: Generate docs and verify build**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
```

- [ ] **Step 7.4: Commit**

```bash
git add server/internal/singbox/
git commit -m "feat: add swagger annotations for singbox domain"
```

---

### Task 8: system endpoints + final verification

**Files:**
- Modify: `server/main.go`

- [ ] **Step 8.1: Add swaggo annotation for health check endpoint**

```go
// healthCheck handles GET /health requests
// @Summary      Health check
// @Description  Returns the server health status
// @Tags         system
// @Produce      json
// @Success      200  {object}  map[string]string  "status ok"
// @Router       /health [get]
```

- [ ] **Step 8.2: Final generation and verification**

```bash
cd /home/kev/work/projects/singbox_ui/server
swag init -g main.go --output internal/docs
go build ./...
golangci-lint run ./...
```

- [ ] **Step 8.3: Commit**

```bash
git add server/main.go server/internal/docs/
git commit -m "feat: add swagger annotations for system endpoints and finalize"
```
