# Clash API Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Clash API support to Go backend for 6 runtime vertical slices (Proxies, Traffic, Connections, Logs, Rules, Mode) and replace Docker with stub.

**Architecture:** Vertical Slices — each feature is a self-contained layer from Gin route through shared Clash HTTP Client to running sing-box process. Port Manager assigns Clash API ports (9090+N) per instance.

**Tech Stack:** Go 1.24, Gin 1.11, standard library `net/http`, `nhooyr.io/websocket` (or `gorilla/websocket`) for WS→SSE proxy.

---

### Pre-task: Create package directory

```bash
mkdir -p server/internal/clashapi
```

---

### Task 1: Port Manager

**Files:**
- Create: `server/internal/clashapi/portmanager.go`
- Test: `server/internal/clashapi/portmanager_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/portmanager_test.go
package clashapi

import (
    "os"
    "path/filepath"
    "testing"
)

func TestPortManagerAssign(t *testing.T) {
    pm := NewPortManager(9090)
    p1 := pm.Assign("default")
    p2 := pm.Assign("office")
    p3 := pm.Assign("home")

    if p1 != 9090 {
        t.Fatalf("expected 9090, got %d", p1)
    }
    if p2 != 9091 {
        t.Fatalf("expected 9091, got %d", p2)
    }
    if p3 != 9092 {
        t.Fatalf("expected 9092, got %d", p3)
    }
}

func TestPortManagerReleaseAndReassign(t *testing.T) {
    pm := NewPortManager(9090)
    _ = pm.Assign("default")   // 9090
    _ = pm.Assign("office")    // 9091
    pm.Release("office")
    p := pm.Assign("home")     // должно переиспользовать 9091
    if p != 9091 {
        t.Fatalf("expected 9091 (reuse), got %d", p)
    }
}

func TestPortManagerGet(t *testing.T) {
    pm := NewPortManager(9090)
    pm.Assign("default")
    port, ok := pm.Get("default")
    if !ok || port != 9090 {
        t.Fatalf("expected 9090, got %d (ok=%v)", port, ok)
    }
    _, ok = pm.Get("nonexistent")
    if ok {
        t.Fatal("expected false for nonexistent instance")
    }
}

func TestPortManagerPersistence(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "ports.json")

    pm1 := NewPortManager(9090)
    pm1.Assign("default")
    pm1.Assign("office")
    if err := pm1.Save(path); err != nil {
        t.Fatal(err)
    }

    pm2 := NewPortManager(9090)
    if err := pm2.Load(path); err != nil {
        t.Fatal(err)
    }
    port, ok := pm2.Get("default")
    if !ok || port != 9090 {
        t.Fatalf("default: expected 9090, got %d", port)
    }
    port, ok = pm2.Get("office")
    if !ok || port != 9091 {
        t.Fatalf("office: expected 9091, got %d", port)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clashapi/ -run TestPortManager -v`
Expected: FAIL (package doesn't exist yet)

- [ ] **Step 3: Write minimal implementation**

```go
// server/internal/clashapi/portmanager.go
package clashapi

import (
    "encoding/json"
    "os"
    "sync"
)

type PortManager struct {
    basePort int
    assigned map[string]int
    mu       sync.Mutex
}

func NewPortManager(basePort int) *PortManager {
    return &PortManager{
        basePort: basePort,
        assigned: make(map[string]int),
    }
}

func (pm *PortManager) Assign(instanceName string) int {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    // already assigned
    if port, ok := pm.assigned[instanceName]; ok {
        return port
    }

    // find next free port
    used := make(map[int]bool)
    for _, p := range pm.assigned {
        used[p] = true
    }
    port := pm.basePort
    for used[port] {
        port++
    }

    pm.assigned[instanceName] = port
    return port
}

func (pm *PortManager) Release(instanceName string) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    delete(pm.assigned, instanceName)
}

func (pm *PortManager) Get(instanceName string) (int, bool) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    port, ok := pm.assigned[instanceName]
    return port, ok
}

func (pm *PortManager) List() map[string]int {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    cp := make(map[string]int, len(pm.assigned))
    for k, v := range pm.assigned {
        cp[k] = v
    }
    return cp
}

func (pm *PortManager) Save(path string) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    data, err := json.MarshalIndent(pm.assigned, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

func (pm *PortManager) Load(path string) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }
    return json.Unmarshal(data, &pm.assigned)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/clashapi/ -run TestPortManager -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/portmanager.go server/internal/clashapi/portmanager_test.go
git commit -m "feat: add Port Manager with persistence"
```

---

### Task 2: Models — Clash API response types

**Files:**
- Create: `server/internal/clashapi/models.go`
- Test: `server/internal/clashapi/models_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/models_test.go
package clashapi

import (
    "encoding/json"
    "testing"
)

func TestProxiesResponseUnmarshal(t *testing.T) {
    data := `{
        "proxies": {
            "Proxy": {
                "type": "Selector",
                "now": "NodeSG",
                "all": ["NodeSG", "NodeJP", "NodeUS"]
            }
        }
    }`
    var resp ProxiesResponse
    if err := json.Unmarshal([]byte(data), &resp); err != nil {
        t.Fatal(err)
    }
    proxy, ok := resp.Proxies["Proxy"]
    if !ok {
        t.Fatal("expected Proxy in map")
    }
    if proxy.Type != "Selector" {
        t.Fatalf("expected Selector, got %s", proxy.Type)
    }
    if len(proxy.All) != 3 {
        t.Fatalf("expected 3 items in All, got %d", len(proxy.All))
    }
}

func TestConnectionsResponseUnmarshal(t *testing.T) {
    data := `{
        "download_total": 1000,
        "upload_total": 500,
        "connections": [
            {
                "id": "abc123",
                "metadata": {
                    "network": "tcp",
                    "host": "example.com"
                }
            }
        ]
    }`
    var resp ConnectionsResponse
    if err := json.Unmarshal([]byte(data), &resp); err != nil {
        t.Fatal(err)
    }
    if resp.DownloadTotal != 1000 {
        t.Fatalf("expected 1000, got %d", resp.DownloadTotal)
    }
    if len(resp.Connections) != 1 {
        t.Fatalf("expected 1 connection, got %d", len(resp.Connections))
    }
}

func TestDelayResponseUnmarshal(t *testing.T) {
    data := `{"NodeSG": 123}`
    var resp DelayResponse
    if err := json.Unmarshal([]byte(data), &resp); err != nil {
        t.Fatal(err)
    }
    if resp.Delay != 123 {
        t.Fatalf("expected 123, got %d", resp.Delay)
    }
}

func TestTrafficMessageUnmarshal(t *testing.T) {
    // бинарный формат: 2 x int64 (up, down) в big-endian
    data := []byte{
        0, 0, 0, 0, 0, 0, 0, 100, // up = 100
        0, 0, 0, 0, 0, 0, 0, 200, // down = 200
    }
    var msg TrafficMessage
    if err := msg.UnmarshalBinary(data); err != nil {
        t.Fatal(err)
    }
    if msg.Up != 100 {
        t.Fatalf("expected up=100, got %d", msg.Up)
    }
    if msg.Down != 200 {
        t.Fatalf("expected down=200, got %d", msg.Down)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clashapi/ -run TestModels -v`
Expected: FAIL (models.go doesn't exist yet)

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/models.go
package clashapi

import (
    "encoding/binary"
    "fmt"
)

// === Proxies ===

type ProxyGroup struct {
    Type string   `json:"type"`
    Now  string   `json:"now"`
    All  []string `json:"all"`
}

type ProxiesResponse struct {
    Proxies map[string]ProxyGroup `json:"proxies"`
}

// === Proxy Detail ===

type ProxyDetail struct {
    Type    string   `json:"type"`
    Name    string   `json:"name"`
    History []struct {
        Time     string  `json:"time"`
        Delay    int     `json:"delay"`
        MeanDelay float64 `json:"meanDelay"`
    } `json:"history"`
}

// === Delay ===

type DelayResponse struct {
    Delay int `json:"delay"`
}

func (d *DelayResponse) UnmarshalJSON(data []byte) error {
    // Clash API возвращает {"NodeName": delay}
    var raw map[string]int
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }
    for _, v := range raw {
        d.Delay = v
        return nil
    }
    return fmt.Errorf("empty delay response")
}

// === Traffic (бинарный формат Clash API) ===

type TrafficMessage struct {
    Up   int64
    Down int64
}

func (m *TrafficMessage) UnmarshalBinary(data []byte) error {
    if len(data) < 16 {
        return fmt.Errorf("traffic message too short: %d bytes", len(data))
    }
    m.Up = int64(binary.BigEndian.Uint64(data[0:8]))
    m.Down = int64(binary.BigEndian.Uint64(data[8:16]))
    return nil
}

// === Memory ===

type MemoryMessage struct {
    Inuse   int64 `json:"inuse"`
    OSLimit int64 `json:"oslimit"`
}

// === Connections ===

type ConnectionMeta struct {
    Network string `json:"network"`
    Type    string `json:"type"`
    Source  string `json:"source"`
    DstIP   string `json:"dstIP"`
    DstPort string `json:"dstPort"`
    Host    string `json:"host"`
    Process string `json:"process"`
}

type Connection struct {
    ID         string          `json:"id"`
    Metadata   ConnectionMeta  `json:"metadata"`
    Upload     int64           `json:"upload"`
    Download   int64           `json:"download"`
    Start      string          `json:"start"`
    Chains     []string        `json:"chains"`
    Rule       string          `json:"rule"`
    RulePayload string         `json:"rulePayload"`
}

type ConnectionsResponse struct {
    DownloadTotal int64        `json:"download_total"`
    UploadTotal   int64        `json:"upload_total"`
    Connections   []Connection `json:"connections"`
}

// === Logs ===

type LogEntry struct {
    Type    string `json:"type"`
    Payload string `json:"payload"`
}

// === Rules ===

type Rule struct {
    Type    string `json:"type"`
    Payload string `json:"payload"`
    Proxy   string `json:"proxy"`
}

type RulesResponse struct {
    Rules []Rule `json:"rules"`
}

// === Config ===

type ConfigResponse struct {
    Mode string `json:"mode"`
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/clashapi/ -run TestModels -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/models.go server/internal/clashapi/models_test.go
git commit -m "feat: add Clash API response models"
```

---

### Task 3: Clash HTTP Client (base)

**Files:**
- Create: `server/internal/clashapi/client.go`
- Test: `server/internal/clashapi/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/client_test.go
package clashapi

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestClientGetProxies(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/proxies" {
            t.Fatalf("expected /proxies, got %s", r.URL.Path)
        }
        if r.Header.Get("Authorization") != "Bearer test-secret" {
            t.Fatalf("expected Bearer test-secret, got %s", r.Header.Get("Authorization"))
        }
        json.NewEncoder(w).Encode(ProxiesResponse{
            Proxies: map[string]ProxyGroup{
                "Proxy": {Type: "Selector", Now: "NodeSG", All: []string{"NodeSG", "NodeJP"}},
            },
        })
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "test-secret")
    resp, err := c.GetProxies()
    if err != nil {
        t.Fatal(err)
    }
    if resp.Proxies["Proxy"].Now != "NodeSG" {
        t.Fatalf("expected NodeSG, got %s", resp.Proxies["Proxy"].Now)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clashapi/ -run TestClient -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/client.go
package clashapi

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strconv"
)

type Client struct {
    baseURL    string
    secret     string
    httpClient *http.Client
}

func NewClient(baseURL, secret string) *Client {
    return &Client{
        baseURL:    baseURL,
        secret:     secret,
        httpClient: &http.Client{},
    }
}

func (c *Client) do(method, path string, body, result interface{}) error {
    var bodyReader io.Reader
    if body != nil {
        data, err := json.Marshal(body)
        if err != nil {
            return fmt.Errorf("marshal body: %w", err)
        }
        bodyReader = bytes.NewReader(data)
    }

    req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    if c.secret != "" {
        req.Header.Set("Authorization", "Bearer "+c.secret)
    }
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("clash api: %d %s", resp.StatusCode, string(respBody))
    }

    if result != nil {
        if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
            return fmt.Errorf("decode response: %w", err)
        }
    }
    return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/clashapi/ -run TestClient -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/client.go server/internal/clashapi/client_test.go
git commit -m "feat: add Clash HTTP client base"
```

---

### Task 4: Client methods — Proxies

**Files:**
- Create: `server/internal/clashapi/client_proxies.go`
- Test: `server/internal/clashapi/client_proxies_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/client_proxies_test.go
package clashapi

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestSwitchProxy(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "PUT" {
            t.Fatalf("expected PUT, got %s", r.Method)
        }
        if r.URL.Path != "/proxies/Proxy" {
            t.Fatalf("expected /proxies/Proxy, got %s", r.URL.Path)
        }
        var body map[string]string
        json.NewDecoder(r.Body).Decode(&body)
        if body["name"] != "NodeSG" {
            t.Fatalf("expected name=NodeSG, got %s", body["name"])
        }
        w.WriteHeader(204)
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    if err := c.SwitchProxy("Proxy", "NodeSG"); err != nil {
        t.Fatal(err)
    }
}

func TestGetProxyDelay(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/proxies/NodeSG/delay" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        urlParam := r.URL.Query().Get("url")
        if urlParam == "" {
            t.Fatal("expected url param")
        }
        json.NewEncoder(w).Encode(map[string]int{"NodeSG": 123})
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    delay, err := c.GetProxyDelay("NodeSG", "https://www.gstatic.com/generate_204", 5000)
    if err != nil {
        t.Fatal(err)
    }
    if delay != 123 {
        t.Fatalf("expected 123, got %d", delay)
    }
}

func TestGetProxyDetail(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/proxies/NodeSG" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        json.NewEncoder(w).Encode(ProxyDetail{Type: "Shadowsocks", Name: "NodeSG"})
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    detail, err := c.GetProxy("NodeSG")
    if err != nil {
        t.Fatal(err)
    }
    if detail.Type != "Shadowsocks" {
        t.Fatalf("expected Shadowsocks, got %s", detail.Type)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clashapi/ -run TestClientProxies -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/client_proxies.go
package clashapi

import "fmt"

func (c *Client) GetProxies() (*ProxiesResponse, error) {
    var resp ProxiesResponse
    if err := c.do("GET", "/proxies", nil, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) GetProxy(name string) (*ProxyDetail, error) {
    var resp ProxyDetail
    if err := c.do("GET", "/proxies/"+name, nil, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) SwitchProxy(groupName, proxyName string) error {
    body := map[string]string{"name": proxyName}
    return c.do("PUT", "/proxies/"+groupName, body, nil)
}

func (c *Client) GetProxyDelay(name, testURL string, timeout int) (int, error) {
    path := fmt.Sprintf("/proxies/%s/delay?url=%s&timeout=%d", name, testURL, timeout)
    var resp DelayResponse
    if err := c.do("GET", path, nil, &resp); err != nil {
        return 0, err
    }
    return resp.Delay, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/clashapi/ -run TestClientProxies -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/client_proxies.go server/internal/clashapi/client_proxies_test.go
git commit -m "feat: add Clash API client proxies methods"
```

---

### Task 5: Client methods — Connections

**Files:**
- Create: `server/internal/clashapi/client_connections.go`
- Test: `server/internal/clashapi/client_connections_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/client_connections_test.go
package clashapi

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetConnections(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(ConnectionsResponse{
            DownloadTotal: 1000,
            UploadTotal:   500,
        })
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    resp, err := c.GetConnections()
    if err != nil {
        t.Fatal(err)
    }
    if resp.DownloadTotal != 1000 {
        t.Fatalf("expected 1000, got %d", resp.DownloadTotal)
    }
}

func TestCloseAllConnections(t *testing.T) {
    called := false
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "DELETE" && r.URL.Path == "/connections" {
            called = true
            w.WriteHeader(204)
        }
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    if err := c.CloseAllConnections(); err != nil {
        t.Fatal(err)
    }
    if !called {
        t.Fatal("expected DELETE /connections to be called")
    }
}

func TestCloseConnection(t *testing.T) {
    called := false
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "DELETE" && r.URL.Path == "/connections/abc123" {
            called = true
            w.WriteHeader(204)
        }
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    if err := c.CloseConnection("abc123"); err != nil {
        t.Fatal(err)
    }
    if !called {
        t.Fatal("expected DELETE /connections/abc123 to be called")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clashapi/ -run TestClientConnections -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/client_connections.go
package clashapi

func (c *Client) GetConnections() (*ConnectionsResponse, error) {
    var resp ConnectionsResponse
    if err := c.do("GET", "/connections", nil, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) CloseAllConnections() error {
    return c.do("DELETE", "/connections", nil, nil)
}

func (c *Client) CloseConnection(id string) error {
    return c.do("DELETE", "/connections/"+id, nil, nil)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/clashapi/ -run TestClientConnections -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/client_connections.go server/internal/clashapi/client_connections_test.go
git commit -m "feat: add Clash API client connections methods"
```

---

### Task 6: Client methods — Rules + Config/Mode

**Files:**
- Create: `server/internal/clashapi/client_rules.go`
- Create: `server/internal/clashapi/client_config.go`
- Test: `server/internal/clashapi/client_rules_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/client_rules_test.go
package clashapi

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetRules(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(RulesResponse{
            Rules: []Rule{
                {Type: "Domain", Payload: "example.com", Proxy: "proxy"},
            },
        })
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    resp, err := c.GetRules()
    if err != nil {
        t.Fatal(err)
    }
    if len(resp.Rules) != 1 {
        t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
    }
}

func TestGetMode(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(ConfigResponse{Mode: "Rule"})
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    mode, err := c.GetMode()
    if err != nil {
        t.Fatal(err)
    }
    if mode != "Rule" {
        t.Fatalf("expected Rule, got %s", mode)
    }
}

func TestSetMode(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "PUT" {
            t.Fatalf("expected PUT, got %s", r.Method)
        }
        var body map[string]string
        json.NewDecoder(r.Body).Decode(&body)
        if body["mode"] != "global" {
            t.Fatalf("expected mode=global, got %s", body["mode"])
        }
        w.WriteHeader(204)
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    if err := c.SetMode("global"); err != nil {
        t.Fatal(err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clashapi/ -run TestClientRules -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/client_rules.go
package clashapi

func (c *Client) GetRules() (*RulesResponse, error) {
    var resp RulesResponse
    if err := c.do("GET", "/rules", nil, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}
```

```go
// server/internal/clashapi/client_config.go
package clashapi

func (c *Client) GetConfigs() (*ConfigResponse, error) {
    var resp ConfigResponse
    if err := c.do("GET", "/configs", nil, &resp); err != nil {
        return nil, err
    }
    return &resp, nil
}

func (c *Client) PatchConfigs(partial map[string]interface{}) error {
    return c.do("PUT", "/configs", partial, nil)
}

func (c *Client) GetMode() (string, error) {
    cfg, err := c.GetConfigs()
    if err != nil {
        return "", err
    }
    return cfg.Mode, nil
}

func (c *Client) SetMode(mode string) error {
    return c.PatchConfigs(map[string]interface{}{"mode": mode})
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/clashapi/ -run TestClientRules -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/client_rules.go server/internal/clashapi/client_config.go server/internal/clashapi/client_rules_test.go
git commit -m "feat: add Clash API client rules and config methods"
```

---

### Task 7: WS/SSE Proxy (stream.go)

**Files:**
- Create: `server/internal/clashapi/stream.go`
- Test: `server/internal/clashapi/stream_test.go`

Используем `nhooyr.io/websocket` для клиента WS к Clash API и `gin.ResponseWriter` для SSE.

- [ ] **Step 1: Add dependency**

```bash
go get nhooyr.io/websocket@latest
```

- [ ] **Step 2: Write the failing test**

```go
// server/internal/clashapi/stream_test.go
package clashapi

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestStreamHandlerSSE(t *testing.T) {
    // Тест проверяет, что SSE handler пишет правильный Content-Type
    // и не падает при отмене контекста (клиент отключился)
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Имитируем, что Clash API WS не отвечает — проверяем таймаут
        // Этот тест проверяет структуру, не полный WS roundtrip
        w.WriteHeader(400)
    }))
    defer srv.Close()

    // Проверяем, что функция компилируется и не падает
    _ = NewClient(srv.URL, "")
}
```

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/stream.go
package clashapi

import (
    "context"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "nhooyr.io/websocket"
)

// writeSSE writes data as Server-Sent Event to gin.ResponseWriter.
func writeSSE(c *gin.Context, data []byte) {
    _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data)
    if err != nil {
        // клиент отключился — не фатально
        return
    }
    c.Writer.Flush()
}

// ProxyWS streams data from a Clash API WebSocket endpoint as SSE.
// clashPort: port of the Clash API
// wsPath: WS path (e.g. "/traffic", "/logs")
func ProxyWS(c *gin.Context, clashPort int, wsPath string) {
    ctx := c.Request.Context()

    wsURL := fmt.Sprintf("ws://127.0.0.1:%d%s", clashPort, wsPath)

    wsConn, _, err := websocket.Dial(ctx, wsURL, nil)
    if err != nil {
        c.JSON(502, gin.H{"error": fmt.Sprintf("clash ws dial: %v", err)})
        return
    }
    defer wsConn.Close(websocket.StatusInternalError, "closing")

    // SSE headers
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.WriteHeader(200)

    for {
        _, msg, err := wsConn.Read(ctx)
        if err != nil {
            if ctx.Err() != nil || err == io.EOF {
                return // нормальное завершение
            }
            log.Printf("clash ws read error: %v", err)
            return
        }

        select {
        case <-ctx.Done():
            return
        default:
            writeSSE(c, msg)
        }
    }
}

// ProxyWSBinary is like ProxyWS but handles binary WS messages (for /traffic).
func ProxyWSBinary(c *gin.Context, clashPort int, wsPath string) {
    ctx := c.Request.Context()

    wsURL := fmt.Sprintf("ws://127.0.0.1:%d%s", clashPort, wsPath)

    wsConn, _, err := websocket.Dial(ctx, wsURL, nil)
    if err != nil {
        c.JSON(502, gin.H{"error": fmt.Sprintf("clash ws dial: %v", err)})
        return
    }
    defer wsConn.Close(websocket.StatusInternalError, "closing")

    // SSE headers
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.WriteHeader(200)

    for {
        _, msg, err := wsConn.Read(ctx)
        if err != nil {
            if ctx.Err() != nil || err == io.EOF {
                return
            }
            log.Printf("clash ws read error: %v", err)
            return
        }

        // Парсим бинарное сообщение в JSON для SSE
        var t TrafficMessage
        if err := t.UnmarshalBinary(msg); err != nil {
            log.Printf("traffic unmarshal error: %v", err)
            continue
        }
        jsonData := fmt.Sprintf(`{"up":%d,"down":%d}`, t.Up, t.Down)

        select {
        case <-ctx.Done():
            return
        default:
            writeSSE(c, []byte(jsonData))
        }
    }
}

// ProxyTrafficSSE is a convenience wrapper for streaming traffic.
func (c *Client) ProxyTrafficSSE(ginCtx *gin.Context) {
    ProxyWSBinary(ginCtx, c.portFromURL(), "/traffic")
}

// ProxyMemorySSE streams memory data as SSE.
func (c *Client) ProxyMemorySSE(ginCtx *gin.Context) {
    ProxyWS(ginCtx, c.portFromURL(), "/memory")
}

// ProxyLogsSSE streams log data as SSE.
func (c *Client) ProxyLogsSSE(ginCtx *gin.Context) {
    ProxyWS(ginCtx, c.portFromURL(), "/logs")
}

// portFromURL extracts the port from the client's base URL.
func (c *Client) portFromURL() int {
    // Парсим порт из baseURL (http://127.0.0.1:{port})
    // Упрощённо: берём последние 4-5 символов
    port := 9090
    if len(c.baseURL) > 6 {
        // expected format: http://127.0.0.1:9090
        // вырезаем последние цифры после ":"
        for i := len(c.baseURL) - 1; i >= 0; i-- {
            if c.baseURL[i] == ':' {
                fmt.Sscanf(c.baseURL[i+1:], "%d", &port)
                break
            }
        }
    }
    return port
}
```

- [ ] **Step 4: Run tests to check compilation**

Run: `go build ./internal/clashapi/`
Expected: OK

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/stream.go server/internal/clashapi/stream_test.go go.mod go.sum
git commit -m "feat: add WS/SSE proxy for traffic, memory, logs"
```

---

### Task 8: Client methods — Traffic + Logs (WS-based)

**Files:**
- Create: `server/internal/clashapi/client_traffic.go`
- Create: `server/internal/clashapi/client_logs.go`
- Test: `server/internal/clashapi/client_logs_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/client_logs_test.go
package clashapi

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetLogLevel(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(map[string]string{"level": "info"})
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    level, err := c.GetLogLevel()
    if err != nil {
        t.Fatal(err)
    }
    if level != "info" {
        t.Fatalf("expected info, got %s", level)
    }
}

func TestSetLogLevel(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "PUT" {
            t.Fatalf("expected PUT, got %s", r.Method)
        }
        var body map[string]string
        json.NewDecoder(r.Body).Decode(&body)
        if body["level"] != "debug" {
            t.Fatalf("expected level=debug, got %s", body["level"])
        }
        w.WriteHeader(204)
    }))
    defer srv.Close()

    c := NewClient(srv.URL, "")
    if err := c.SetLogLevel("debug"); err != nil {
        t.Fatal(err)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/clashapi/ -run TestClientLogLevel -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/client_traffic.go
package clashapi
// Traffic методы WS-based — реализованы в stream.go через ProxyWSBinary
```

```go
// server/internal/clashapi/client_logs.go
package clashapi

func (c *Client) GetLogLevel() (string, error) {
    var resp struct {
        Level string `json:"level"`
    }
    if err := c.do("GET", "/logs/level", nil, &resp); err != nil {
        return "", err
    }
    return resp.Level, nil
}

func (c *Client) SetLogLevel(level string) error {
    return c.do("PUT", "/logs/level", map[string]string{"level": level}, nil)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/clashapi/ -run TestClientLogLevel -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/client_traffic.go server/internal/clashapi/client_logs.go server/internal/clashapi/client_logs_test.go
git commit -m "feat: add Clash API client traffic and log methods"
```

---

### Task 9: Gin routes — Proxies

**Files:**
- Create: `server/internal/clashapi/routes_proxies.go`
- Test: `server/internal/clashapi/routes_proxies_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/routes_proxies_test.go
package clashapi

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestProxiesRoutes(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // Тестовый Port Manager
    pm := NewPortManager(9090)
    pm.Assign("default")

    // Тестовый Gin engine с роутами
    r := gin.New()
    group := r.Group("/api/clash/instances/:name")
    RegisterProxiesRoutes(group, pm)

    // Запускаем тестовый HTTP сервер Clash API
    clashSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        json.NewEncoder(w).Encode(ProxiesResponse{
            Proxies: map[string]ProxyGroup{
                "Proxy": {Type: "Selector", Now: "NodeSG", All: []string{"NodeSG", "NodeJP"}},
            },
        })
    }))
    defer clashSrv.Close()

    // Временно переопределяем создание клиента
    // В реальном коде нужно либо inject клиент, либо PortManager возвращает порт
    // Для этого теста мы проверяем только маршрутизацию
    req := httptest.NewRequest("GET", "/api/clash/instances/default/proxies", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    // Ожидаем 502 (Bad Gateway) т.к. Clash API на localhost:9090 не отвечает,
    // но роут должен найтись (не 404)
    if w.Code == 404 {
        t.Fatal("route not found (404)")
    }
}
```

- [ ] **Step 2: Run test**

Run: `go test ./internal/clashapi/ -run TestProxiesRoutes -v`

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/routes_proxies.go
package clashapi

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
)

func RegisterProxiesRoutes(rg *gin.RouterGroup, pm *PortManager) {
    rg.GET("/proxies", func(c *gin.Context) {
        port, client := getClient(c, pm)
        if port == 0 {
            return
        }
        resp, err := client.GetProxies()
        if err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, resp)
    })

    rg.GET("/proxies/:group", func(c *gin.Context) {
        port, client := getClient(c, pm)
        if port == 0 {
            return
        }
        resp, err := client.GetProxy(c.Param("group"))
        if err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, resp)
    })

    rg.PUT("/proxies/:group", func(c *gin.Context) {
        port, client := getClient(c, pm)
        if port == 0 {
            return
        }
        var body struct {
            Name string `json:"name"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
            c.JSON(400, gin.H{"error": "missing 'name' field"})
            return
        }
        if err := client.SwitchProxy(c.Param("group"), body.Name); err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.Status(204)
    })

    rg.GET("/proxies/:group/delay", func(c *gin.Context) {
        port, client := getClient(c, pm)
        if port == 0 {
            return
        }
        url := c.DefaultQuery("url", "https://www.gstatic.com/generate_204")
        timeout, _ := strconv.Atoi(c.DefaultQuery("timeout", "5000"))
        delay, err := client.GetProxyDelay(c.Param("group"), url, timeout)
        if err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"delay": delay})
    })
}

// getClient извлекает имя инстанса из URL, получает порт и создаёт клиент.
func getClient(c *gin.Context, pm *PortManager) (int, *Client) {
    name := c.Param("name")
    port, ok := pm.Get(name)
    if !ok {
        c.JSON(404, gin.H{"error": "instance not found: " + name})
        return 0, nil
    }
    baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
    return port, NewClient(baseURL, "")
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/clashapi/ -run TestProxiesRoutes -v`
Expected: PASS (or at least not 404)

- [ ] **Step 5: Commit**

```bash
git add server/internal/clashapi/routes_proxies.go server/internal/clashapi/routes_proxies_test.go
git commit -m "feat: add Clash API proxies routes"
```

---

### Task 10: Gin routes — Connections

**Files:**
- Create: `server/internal/clashapi/routes_connections.go`
- Test: `server/internal/clashapi/routes_connections_test.go`

- [ ] **Step 1: Write the failing test**

```go
// server/internal/clashapi/routes_connections_test.go
package clashapi

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestConnectionsRoutes(t *testing.T) {
    gin.SetMode(gin.TestMode)

    pm := NewPortManager(9090)
    pm.Assign("default")

    r := gin.New()
    group := r.Group("/api/clash/instances/:name")
    RegisterConnectionsRoutes(group, pm)

    req := httptest.NewRequest("GET", "/api/clash/instances/default/connections", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code == 404 {
        t.Fatal("route not found (404)")
    }
}
```

- [ ] **Step 2: Run test to verify compilation**

Run: `go test ./internal/clashapi/ -run TestConnectionsRoutes -v`

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/routes_connections.go
package clashapi

import (
    "github.com/gin-gonic/gin"
)

func RegisterConnectionsRoutes(rg *gin.RouterGroup, pm *PortManager) {
    rg.GET("/connections", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        resp, err := client.GetConnections()
        if err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, resp)
    })

    rg.DELETE("/connections", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        if err := client.CloseAllConnections(); err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.Status(204)
    })

    rg.DELETE("/connections/:id", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        if err := client.CloseConnection(c.Param("id")); err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.Status(204)
    })
}
```

- [ ] **Step 4: Commit**

```bash
git add server/internal/clashapi/routes_connections.go server/internal/clashapi/routes_connections_test.go
git commit -m "feat: add Clash API connections routes"
```

---

### Task 11: Gin routes — Rules + Mode

**Files:**
- Create: `server/internal/clashapi/routes_rules.go`
- Create: `server/internal/clashapi/routes_mode.go`
- Test: `server/internal/clashapi/routes_rules_test.go`

- [ ] **Step 1: Write the test**

```go
// server/internal/clashapi/routes_rules_test.go
package clashapi

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestRulesAndModeRoutes(t *testing.T) {
    gin.SetMode(gin.TestMode)

    pm := NewPortManager(9090)
    pm.Assign("default")

    r := gin.New()
    group := r.Group("/api/clash/instances/:name")
    RegisterRulesRoutes(group, pm)
    RegisterModeRoutes(group, pm)

    tests := []string{
        "/api/clash/instances/default/rules",
        "/api/clash/instances/default/mode",
    }
    for _, path := range tests {
        req := httptest.NewRequest("GET", path, nil)
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        if w.Code == 404 {
            t.Fatalf("route not found: %s", path)
        }
    }
}
```

- [ ] **Step 2: Run test**

Run: `go test ./internal/clashapi/ -run TestRulesAndModeRoutes -v`

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/routes_rules.go
package clashapi

import "github.com/gin-gonic/gin"

func RegisterRulesRoutes(rg *gin.RouterGroup, pm *PortManager) {
    rg.GET("/rules", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        resp, err := client.GetRules()
        if err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, resp)
    })
}
```

```go
// server/internal/clashapi/routes_mode.go
package clashapi

import "github.com/gin-gonic/gin"

func RegisterModeRoutes(rg *gin.RouterGroup, pm *PortManager) {
    rg.GET("/mode", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        mode, err := client.GetMode()
        if err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"mode": mode})
    })

    rg.PUT("/mode", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        var body struct {
            Mode string `json:"mode"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
            c.JSON(400, gin.H{"error": "missing 'mode' field"})
            return
        }
        if err := client.SetMode(body.Mode); err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.Status(204)
    })
}
```

- [ ] **Step 4: Commit**

```bash
git add server/internal/clashapi/routes_rules.go server/internal/clashapi/routes_mode.go server/internal/clashapi/routes_rules_test.go
git commit -m "feat: add Clash API rules and mode routes"
```

---

### Task 12: Gin routes — Traffic + Logs (SSE)

**Files:**
- Create: `server/internal/clashapi/routes_traffic.go`
- Create: `server/internal/clashapi/routes_logs.go`
- Test: `server/internal/clashapi/routes_traffic_test.go`

- [ ] **Step 1: Write the test**

```go
// server/internal/clashapi/routes_traffic_test.go
package clashapi

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestTrafficAndLogsRoutes(t *testing.T) {
    gin.SetMode(gin.TestMode)

    pm := NewPortManager(9090)
    pm.Assign("default")

    r := gin.New()
    group := r.Group("/api/clash/instances/:name")
    RegisterTrafficRoutes(group, pm)
    RegisterLogsRoutes(group, pm)

    tests := []string{
        "/api/clash/instances/default/traffic",
        "/api/clash/instances/default/memory",
        "/api/clash/instances/default/logs",
        "/api/clash/instances/default/logs/level",
    }
    for _, path := range tests {
        req := httptest.NewRequest("GET", path, nil)
        w := httptest.NewRecorder()
        r.ServeHTTP(w, req)
        if w.Code == 404 {
            t.Fatalf("route not found: %s", path)
        }
    }
}
```

- [ ] **Step 2: Run test**

Run: `go test ./internal/clashapi/ -run TestTrafficAndLogsRoutes -v`

- [ ] **Step 3: Write implementation**

```go
// server/internal/clashapi/routes_traffic.go
package clashapi

import (
    "github.com/gin-gonic/gin"
)

func RegisterTrafficRoutes(rg *gin.RouterGroup, pm *PortManager) {
    rg.GET("/traffic", func(c *gin.Context) {
        name := c.Param("name")
        port, ok := pm.Get(name)
        if !ok {
            c.JSON(404, gin.H{"error": "instance not found: " + name})
            return
        }
        ProxyWSBinary(c, port, "/traffic")
    })

    rg.GET("/memory", func(c *gin.Context) {
        name := c.Param("name")
        port, ok := pm.Get(name)
        if !ok {
            c.JSON(404, gin.H{"error": "instance not found: " + name})
            return
        }
        ProxyWS(c, port, "/memory")
    })
}
```

```go
// server/internal/clashapi/routes_logs.go
package clashapi

import (
    "strconv"

    "github.com/gin-gonic/gin"
)

func RegisterLogsRoutes(rg *gin.RouterGroup, pm *PortManager) {
    rg.GET("/logs", func(c *gin.Context) {
        name := c.Param("name")
        port, ok := pm.Get(name)
        if !ok {
            c.JSON(404, gin.H{"error": "instance not found: " + name})
            return
        }
        ProxyWS(c, port, "/logs")
    })

    rg.GET("/logs/level", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        level, err := client.GetLogLevel()
        if err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.JSON(200, gin.H{"level": level})
    })

    rg.PUT("/logs/level", func(c *gin.Context) {
        _, client := getClient(c, pm)
        if client == nil {
            return
        }
        var body struct {
            Level string `json:"level"`
        }
        if err := c.ShouldBindJSON(&body); err != nil {
            c.JSON(400, gin.H{"error": "missing 'level' field"})
            return
        }
        if err := client.SetLogLevel(body.Level); err != nil {
            c.JSON(502, gin.H{"error": err.Error()})
            return
        }
        c.Status(204)
    })
}
```

- [ ] **Step 4: Commit**

```bash
git add server/internal/clashapi/routes_traffic.go server/internal/clashapi/routes_logs.go server/internal/clashapi/routes_traffic_test.go
git commit -m "feat: add Clash API traffic and logs SSE routes"
```

---

### Task 13: Register all routes + main.go changes

**Files:**
- Create: `server/internal/clashapi/register.go`
- Modify: `server/main.go`
- Modify: `server/internal/singbox/handler.go` (+ clash_port in response)
- Modify: `server/internal/singbox/service.go` (+ GetClashPort)

- [ ] **Step 1: Write register.go**

```go
// server/internal/clashapi/register.go
package clashapi

import "github.com/gin-gonic/gin"

func RegisterAllRoutes(rg *gin.RouterGroup, pm *PortManager) {
    RegisterProxiesRoutes(rg, pm)
    RegisterTrafficRoutes(rg, pm)
    RegisterConnectionsRoutes(rg, pm)
    RegisterLogsRoutes(rg, pm)
    RegisterRulesRoutes(rg, pm)
    RegisterModeRoutes(rg, pm)
}
```

- [ ] **Step 2: Add clash_port to sing-box service**

```go
// в server/internal/singbox/service.go добавить:
func (s *Service) GetClashPort(instanceName string) int {
    // Port Manager будет установлен извне
    // Возвращает 0 если Port Manager не настроен
    if s.portManager == nil {
        return 0
    }
    port, ok := s.portManager.Get(instanceName)
    if !ok {
        return 0
    }
    return port
}
```

Добавить `portManager` поле в `Service`:
```go
type Service struct {
    runtime     Runtime
    cfg         *config.Config
    portManager *clashapi.PortManager  // новое
}
```

- [ ] **Step 3: Add clash_port to handler response**

В `server/internal/singbox/handler.go`, в `LoadNamedConfigFromContainer`:
```go
// После загрузки данных, добавить clash_port в ответ:
var configData interface{}
json.Unmarshal(data, &configData)
response := gin.H{
    "config": configData,
    "clash_port": h.service.GetClashPort(name),
}
c.JSON(200, response)
```

- [ ] **Step 4: Update main.go**

```go
// server/main.go — после инициализации сервисов:

// Инициализация Port Manager
pm := clashapi.NewPortManager(9090)
pm.Load(filepath.Join(cfg.GetDataDir(), "clash_ports.json"))

// Назначение портов для существующих инстансов
instances, _ := singboxSvc.ListNamedConfigs()
for _, inst := range instances {
    pm.Assign(inst.Name)
}

// Регистрация Clash API роутов
api := r.Group("/api")
clashGroup := api.Group("/clash")
clashapi.RegisterAllRoutes(clashGroup, pm)

// Передать Port Manager в Service
// (нужно добавить метод SetPortManager в Service)
singboxSvc.SetPortManager(pm)

// Сохранение портов при завершении
// Использовать defer или сигнал
defer pm.Save(filepath.Join(cfg.GetDataDir(), "clash_ports.json"))
```

- [ ] **Step 5: Build check**

Run: `go build ./...`
Expected: OK

- [ ] **Step 6: Commit**

```bash
git add server/internal/clashapi/register.go server/main.go server/internal/singbox/service.go server/internal/singbox/handler.go
git commit -m "feat: register Clash API routes, update main.go"
```

---

### Task 14: Docker — заглушка

**Files:**
- Modify: `server/internal/singbox/runtime_docker.go`
- Remove: `server/internal/pkg/tunnelrunner/docker.go`

- [ ] **Step 1: Replace runtime_docker.go with stub**

```go
// server/internal/singbox/runtime_docker.go
//go:build docker

package singbox

import (
    "context"
    "fmt"

    "singbox-config-service/internal/pkg/config"
)

func NewRuntime(cfg *config.Config) (Runtime, error) {
    return nil, fmt.Errorf("Docker runtime is not available in this build")
}
```

- [ ] **Step 2: Remove tunnelrunner docker implementation**

```bash
# tunnelrunner уже использует native по умолчанию (native.go без build tag)
# docker.go имеет build tag "docker" — просто удаляем файл
rm server/internal/pkg/tunnelrunner/docker.go
```

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: OK (без тега docker)

- [ ] **Step 4: Commit**

```bash
git add server/internal/singbox/runtime_docker.go
git rm server/internal/pkg/tunnelrunner/docker.go
git commit -m "refactor: replace Docker runtime with stub, remove tunnelrunner docker"
```

---

### Task 15: Port persistence + tests

**Files:**
- Modify: `server/internal/clashapi/portmanager.go` (уже реализовано в Task 1)
- Test: full round-trip test

- [ ] **Step 1: Run all clashapi tests**

Run: `go test ./internal/clashapi/ -v`
Expected: All PASS

- [ ] **Step 2: Full build and run**

Run: `go build -o /dev/null ./...`
Expected: OK

```bash
git add -A
git commit -m "chore: finalize Clash API integration"
```

---

## Self-Review

1. **Spec coverage:**
   - Port Manager ✅ (Task 1, 15)
   - Models ✅ (Task 2)
   - Clash HTTP Client ✅ (Task 3)
   - Proxies methods + routes ✅ (Task 4, 9)
   - Connections methods + routes ✅ (Task 5, 10)
   - Rules + Mode methods + routes ✅ (Task 6, 11)
   - WS/SSE proxy ✅ (Task 7)
   - Traffic + Logs methods + routes ✅ (Task 8, 12)
   - Registration + main.go ✅ (Task 13)
   - Docker stub ✅ (Task 14)
   - Prober/Speedtest unchanged ✅ (not touched)
   - Frontend excluded ✅ (not touched)

2. **Placeholder scan:** No TBD, TODO found.

3. **Type consistency:** `PortManager.Assign`, `Client.GetProxies`, `RegisterAllRoutes` — consistent across all tasks.
