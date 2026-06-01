package speedtest

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
	"strings"
	"sync"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

const (
	speedTestLatencyURL  = "http://www.gstatic.com/generate_204"
	speedTestDownloadURL = "https://speed.cloudflare.com/__down?bytes=10000000"
	speedTestDuration    = 10 * time.Second

	speedTestContainerName = "sing-box-speedtest"
	containerConfigDir     = "/etc/sing-box"
	containerDataDir       = "/var/lib/sing-box"
	singBoxImage           = "ghcr.io/sagernet/sing-box:v1.13.5"
)

type Service struct {
	docker       ContainerManager
	cfg          *config.Config
	nodeProvider NodeProvider
	resultSaver  SpeedTestResultSaver
	state        *SpeedTestState
	mu           sync.Mutex
	cancel       context.CancelFunc
}

func NewService(docker ContainerManager, cfg *config.Config) *Service {
	return &Service{
		docker: docker,
		cfg:    cfg,
		state:  &SpeedTestState{},
	}
}

func (s *Service) WithNodeProvider(np NodeProvider) *Service {
	s.nodeProvider = np
	return s
}

func (s *Service) WithResultSaver(rs SpeedTestResultSaver) *Service {
	s.resultSaver = rs
	return s
}

func (s *Service) StartSpeedTest() error {
	s.mu.Lock()
	if s.state.Running {
		s.mu.Unlock()
		return fmt.Errorf("speed test already running")
	}
	if s.nodeProvider == nil {
		s.mu.Unlock()
		return fmt.Errorf("node provider not configured")
	}
	allNodes := s.nodeProvider.GetAllNodes()
	if len(allNodes) == 0 {
		s.mu.Unlock()
		return fmt.Errorf("no nodes to test")
	}

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.state = &SpeedTestState{
		Running: true,
	}
	s.mu.Unlock()

	go s.runSpeedTest(ctx, cancel, allNodes)
	return nil
}

func (s *Service) GetSpeedTestState() *SpeedTestState {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *s.state
	return &cp
}

func (s *Service) StopSpeedTest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Service) runSpeedTest(ctx context.Context, cancel context.CancelFunc, nodes []types.ProxyNode) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[speedtest] PANIC: %v", r)
		}
		s.cleanupContainer()
		s.mu.Lock()
		s.state.Running = false
		if s.state.Status == "testing" {
			s.state.Status = "completed"
		}
		if s.cancel != nil {
			cancel()
			s.cancel = nil
		}
		s.mu.Unlock()
	}()

	total := len(nodes)
	for i, n := range nodes {
		select {
		case <-ctx.Done():
			return
		default:
		}

		tag := nodeOutboundTag(&n)
		s.mu.Lock()
		state := s.state
		state.Tag = tag
		state.Status = "testing"
		state.Progress = i * 100 / total
		s.mu.Unlock()

		latency, speed, dlErr, err := s.testOneNode(ctx, &n, tag)

		s.mu.Lock()
		state.Progress = (i + 1) * 100 / total
		if err != nil {
			state.Status = "failed"
			state.Error = err.Error()
		} else {
			state.Status = "ok"
			state.LatencyMs = latency
			state.DownloadSpeed = speed
			if dlErr != "" {
				state.Error = "download: " + dlErr
			}
		}
		s.mu.Unlock()

		if s.resultSaver != nil {
			online := err == nil
			_ = s.resultSaver.SaveSpeedTestResults([]types.SpeedTestUpdate{
				{
					Tag:       tag,
					Latency:   latency,
					SpeedKBps: speed,
					Online:    online,
					LastProbe: time.Now().Format("2006-01-02 15:04:05"),
				},
			})
		}
	}
}

func nodeOutboundTag(n *types.ProxyNode) string {
	if n.Outbound != nil {
		if t, ok := n.Outbound["tag"].(string); ok && t != "" {
			return t
		}
	}
	return types.SanitizeTag(n.Protocol, n.Address, n.Port)
}

func (s *Service) testOneNode(ctx context.Context, node *types.ProxyNode, tag string) (int64, float64, string, error) {
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

	dir := filepath.Join(s.cfg.GetSingboxDir(), "speedtest")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, 0, "", err
	}
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, cfgBytes, 0644); err != nil {
		return 0, 0, "", err
	}
	defer os.Remove(cfgPath)

	hostConfigPath := cfgPath
	if resolved, err := s.cfg.ResolveHostConfigDir(cfgPath); err == nil {
		hostConfigPath = resolved
	}

	s.cleanupContainer()

	containerConfig := map[string]interface{}{
		"Image": singBoxImage,
		"Cmd":   []string{"-D", containerDataDir, "-C", containerConfigDir + "/", "run"},
	}
	hostConfig := map[string]interface{}{
		"Binds":       []string{hostConfigPath + ":" + containerConfigDir + "/config.json:ro"},
		"NetworkMode": "host",
		"CapAdd":      []string{"NET_ADMIN"},
	}

	var id string
	for attempt := 0; attempt < 5; attempt++ {
		id, err = s.docker.ContainerCreate(ctx, containerConfig, hostConfig, speedTestContainerName)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "already in use") && !strings.Contains(err.Error(), "Conflict") {
			return 0, 0, "", fmt.Errorf("container create: %w", err)
		}
		s.cleanupContainer()
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return 0, 0, "", fmt.Errorf("container create after retries: %w", err)
	}

	if err := s.docker.ContainerStart(ctx, id); err != nil {
		_ = s.docker.ContainerRemove(ctx, id, true)
		return 0, 0, "", fmt.Errorf("container start: %w", err)
	}

	defer func() {
		s.cleanupContainer()
	}()

	if err := waitProxyReady(ctx, port, 10*time.Second); err != nil {
		logs := s.getContainerLogs()
		return 0, 0, "", fmt.Errorf("proxy not ready (port %d): %s", port, logs)
	}

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := newProxyClient(proxyURL, 10*time.Second)

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

func (s *Service) cleanupContainer() {
	_ = s.docker.ContainerRemove(context.Background(), speedTestContainerName, true)
}

func (s *Service) getContainerLogs() string {
	return ""
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

func buildSpeedTestConfig(node *types.ProxyNode, tag string, port int) map[string]interface{} {
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
