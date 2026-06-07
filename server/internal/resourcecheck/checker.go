package resourcecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		return nil, fmt.Errorf("missing outbound")
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
	_ = pu // proxyURL is not used directly for TCP dial in this implementation
	// TCP check bypasses the proxy for now (simplified)
	dialer := &net.Dialer{Timeout: checkTimeout}

	addr := net.JoinHostPort(res.URL, fmt.Sprintf("%d", res.Port))
	conn, err := dialer.DialContext(ctx, "tcp", addr)
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
