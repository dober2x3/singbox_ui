package services

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
	"sync"
	"time"
)

// Proxy speed test: serially start temporary sing-box containers, test node latency and download speed via SOCKS/HTTP proxy

const (
	speedTestLatencyURL  = "http://www.gstatic.com/generate_204"
	speedTestDownloadURL = "https://speed.cloudflare.com/__down?bytes=10000000"
	speedTestDuration    = 10 * time.Second
)

// SpeedTestResult single node speed test result
type SpeedTestResult struct {
	Tag       string  `json:"tag"`
	Name      string  `json:"name"`
	Status    string  `json:"status"` // pending|testing|ok|failed
	LatencyMs int64   `json:"latency_ms"`
	SpeedKBps float64 `json:"speed_kbps"`
	Error     string  `json:"error,omitempty"`
	TestedAt  string  `json:"tested_at,omitempty"`
}

// SpeedTestState global speed test state
type SpeedTestState struct {
	Running   bool                        `json:"running"`
	Total     int                         `json:"total"`
	Done      int                         `json:"done"`
	Current   string                      `json:"current,omitempty"`
	StartedAt string                      `json:"started_at,omitempty"`
	Results   map[string]*SpeedTestResult `json:"results"`
}

var (
	speedTestMu     sync.Mutex
	speedTestState  = &SpeedTestState{Results: map[string]*SpeedTestResult{}}
	speedTestCancel context.CancelFunc
)

// GetSpeedTestState returns a state snapshot
func GetSpeedTestState() *SpeedTestState {
	speedTestMu.Lock()
	defer speedTestMu.Unlock()
	cp := *speedTestState
	cp.Results = make(map[string]*SpeedTestResult, len(speedTestState.Results))
	for k, v := range speedTestState.Results {
		r := *v
		cp.Results[k] = &r
	}
	return &cp
}

// StopSpeedTest cancels a running speed test
func StopSpeedTest() {
	speedTestMu.Lock()
	defer speedTestMu.Unlock()
	if speedTestCancel != nil {
		speedTestCancel()
	}
}

// StartSpeedTest starts a serial speed test (all subscription nodes)
func StartSpeedTest() error {
	speedTestMu.Lock()
	if speedTestState.Running {
		speedTestMu.Unlock()
		return fmt.Errorf("speed test already running")
	}
	allNodes, err := GetAllNodes()
	if err != nil {
		speedTestMu.Unlock()
		return err
	}
	if len(allNodes) == 0 {
		speedTestMu.Unlock()
		return fmt.Errorf("no nodes")
	}

	// Release leftover cancel from previous round (if any)
	if speedTestCancel != nil {
		speedTestCancel()
		speedTestCancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	speedTestCancel = cancel
	speedTestState = &SpeedTestState{
		Running:   true,
		Total:     len(allNodes),
		StartedAt: time.Now().Format(time.RFC3339),
		Results:   make(map[string]*SpeedTestResult, len(allNodes)),
	}
	for _, n := range allNodes {
		tag := nodeOutboundTag(n)
		speedTestState.Results[tag] = &SpeedTestResult{
			Tag: tag, Name: n.Name, Status: "pending",
		}
	}
	speedTestMu.Unlock()

	go runSpeedTest(ctx, cancel, allNodes)
	return nil
}

func nodeOutboundTag(n ProxyNode) string {
	if n.Outbound != nil {
		if t, ok := n.Outbound["tag"].(string); ok && t != "" {
			return t
		}
	}
	return SanitizeTag(n.Protocol, n.Address, n.Port)
}

func runSpeedTest(ctx context.Context, cancel context.CancelFunc, nodes []ProxyNode) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[speedtest] PANIC: %v", r)
		}
		// Ensure temporary container is cleaned up at the end
		if ds := GetDockerService(); ds != nil {
			_ = ds.StopSpeedTestContainer()
		}
		speedTestMu.Lock()
		speedTestState.Running = false
		speedTestState.Current = ""
		// Release this round's own cancel (if global still points to it)
		if speedTestCancel != nil {
			// Call cancel to release context-associated resources (idempotent)
			cancel()
			speedTestCancel = nil
		}
		// Only persist completed nodes (ok/failed), avoid writing cancelled pending/testing nodes
		// as online=false + latency=0, which would overwrite previous valid data
		updates := make([]SpeedTestUpdate, 0, len(speedTestState.Results))
		for _, r := range speedTestState.Results {
			if r.Status != "ok" && r.Status != "failed" {
				continue
			}
			updates = append(updates, SpeedTestUpdate{
				Tag:       r.Tag,
				Latency:   r.LatencyMs,
				SpeedKBps: r.SpeedKBps,
				Online:    r.Status == "ok",
				LastProbe: r.TestedAt,
			})
		}
		speedTestMu.Unlock()
		if len(updates) > 0 {
			if err := UpdateSpeedTestResults(updates); err != nil {
				log.Printf("[speedtest] persist results: %v", err)
			}
		}
	}()

	for _, n := range nodes {
		select {
		case <-ctx.Done():
			return
		default:
		}
		tag := nodeOutboundTag(n)
		speedTestMu.Lock()
		speedTestState.Current = n.Name
		if r, ok := speedTestState.Results[tag]; ok {
			r.Status = "testing"
		}
		speedTestMu.Unlock()

		latency, speed, dlErr, err := testOneNode(ctx, n, tag)

		speedTestMu.Lock()
		r := speedTestState.Results[tag]
		if r == nil {
			r = &SpeedTestResult{Tag: tag, Name: n.Name}
			speedTestState.Results[tag] = r
		}
		r.TestedAt = time.Now().Format("2006-01-02 15:04:05")
		if err != nil {
			r.Status = "failed"
			r.Error = err.Error()
		} else {
			r.Status = "ok"
			r.LatencyMs = latency
			r.SpeedKBps = speed
			if dlErr != "" {
				r.Error = "download: " + dlErr
			}
		}
		speedTestState.Done++
		speedTestMu.Unlock()
	}
}

// testOneNode returns: latencyMs, speedKBps, downloadErrMsg (non-fatal), fatalErr
func testOneNode(ctx context.Context, node ProxyNode, tag string) (int64, float64, string, error) {
	if node.Outbound == nil {
		return 0, 0, "", fmt.Errorf("missing outbound")
	}

	port, err := pickFreePort()
	if err != nil {
		return 0, 0, "", fmt.Errorf("pick port: %w", err)
	}

	cfg := buildSpeedTestConfig(node, tag, port)
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return 0, 0, "", err
	}

	dir := filepath.Join(singboxDir, "speedtest")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, 0, "", err
	}
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, cfgBytes, 0644); err != nil {
		return 0, 0, "", err
	}
	defer os.Remove(cfgPath)

	ds := GetDockerService()
	if ds == nil {
		return 0, 0, "", fmt.Errorf("docker not initialized")
	}

	if err := ds.StartSpeedTestContainer(dir); err != nil {
		return 0, 0, "", fmt.Errorf("container start: %w", err)
	}
	defer ds.StopSpeedTestContainer()

	if err := waitProxyReady(ctx, port, 10*time.Second); err != nil {
		logs := ds.GetSpeedTestContainerLogs()
		return 0, 0, "", fmt.Errorf("proxy not ready (port %d): %s", port, logs)
	}

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := newProxyClient(proxyURL, 10*time.Second)

	// Latency: take minimum of 3 attempts
	var minLatency int64 = -1
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return 0, 0, "", ctx.Err()
		default:
		}
		t0 := time.Now()
		req, _ := http.NewRequestWithContext(ctx, "GET", speedTestLatencyURL, nil)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		ms := time.Since(t0).Milliseconds()
		if minLatency < 0 || ms < minLatency {
			minLatency = ms
		}
	}
	if minLatency < 0 {
		return 0, 0, "", fmt.Errorf("latency probe failed")
	}

	// Download throughput: 10 second limit. Download failure is non-fatal: keep latency data, speed=0, log reason
	dlClient := newProxyClient(proxyURL, speedTestDuration+5*time.Second)
	dlCtx, dlCancel := context.WithTimeout(ctx, speedTestDuration)
	defer dlCancel()
	req, _ := http.NewRequestWithContext(dlCtx, "GET", speedTestDownloadURL, nil)
	t0 := time.Now()
	resp, err := dlClient.Do(req)
	if err != nil {
		return minLatency, 0, err.Error(), nil
	}
	defer resp.Body.Close()
	n, _ := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(t0).Seconds()
	speed := 0.0
	if elapsed > 0 && n > 0 {
		speed = float64(n) / 1024.0 / elapsed
	}
	return minLatency, speed, "", nil
}

func pickFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitProxyReady(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

func newProxyClient(proxyURL string, timeout time.Duration) *http.Client {
	pu, _ := url.Parse(proxyURL)
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(pu),
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: 8 * time.Second,
		},
	}
}

func buildSpeedTestConfig(node ProxyNode, tag string, port int) map[string]interface{} {
	outbound := make(map[string]interface{}, len(node.Outbound)+1)
	for k, v := range node.Outbound {
		outbound[k] = v
	}
	outbound["tag"] = tag

	// Global proxy mode: DNS and all traffic go through the proxy node
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
				"tag":         "speedtest-in",
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
