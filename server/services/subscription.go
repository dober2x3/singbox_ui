package services

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ProxyNode proxy node information
type ProxyNode struct {
	Name     string                 `json:"name"`
	Protocol string                 `json:"protocol"`
	Address  string                 `json:"address"`
	Port     int                    `json:"port"`
	Settings map[string]interface{} `json:"settings"`
	Outbound map[string]interface{} `json:"outbound"` // sing-box format outbound config
	// Speed test related fields
	Latency     int64  `json:"latency,omitempty"`      // latency (ms)
	Online      bool   `json:"online,omitempty"`       // whether online
	LastProbe   string `json:"last_probe,omitempty"`   // last probe time
	SuccessRate int    `json:"success_rate,omitempty"` // success rate (0-100)
	SpeedKBps   float64 `json:"speed_kbps,omitempty"`  // proxy download speed KB/s
}

// decodeBase64 decodes a Base64 string, auto-handles padding and URL-safe encoding
func decodeBase64(s string) ([]byte, error) {
	// Remove possible whitespace
	s = strings.TrimSpace(s)

	// Add padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Try URL-safe encoding
	decoded, err := base64.URLEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}

	// Try standard encoding
	decoded, err = base64.StdEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}

	// Try RawURLEncoding (no padding)
	s = strings.TrimRight(s, "=")
	decoded, err = base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}

	// Try RawStdEncoding (no padding)
	return base64.RawStdEncoding.DecodeString(s)
}

// VMess node configuration
type VMess struct {
	V    string `json:"v"`
	PS   string `json:"ps"`
	Add  string `json:"add"`
	Port string `json:"port"`
	ID   string `json:"id"`
	Aid  string `json:"aid"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	Path string `json:"path"`
	TLS  string `json:"tls"`
	SNI  string `json:"sni"`
}

// ResolveUserAgent resolves User-Agent, supports predefined names and custom values
func ResolveUserAgent(ua string) string {
	if ua == "" {
		return PredefinedUserAgents["default"]
	}
	// First check if it's a predefined name
	if predefined, ok := PredefinedUserAgents[ua]; ok {
		return predefined
	}
	// Otherwise use as custom UA directly
	return ua
}

// FetchSubscription fetches subscription content
var blockedSubscriptionPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("ff00::/8"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func allowInsecureSubscriptionTLS() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("SUBSCRIPTION_INSECURE_TLS")))
	return value == "1" || value == "true" || value == "yes"
}

func isPublicSubscriptionAddr(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range blockedSubscriptionPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

func validateSubscriptionHost(host string) error {
	normalizedHost := strings.Trim(strings.TrimSpace(host), "[]")
	if normalizedHost == "" {
		return fmt.Errorf("subscription URL host is empty")
	}
	if strings.EqualFold(normalizedHost, "localhost") {
		return fmt.Errorf("subscription host localhost is not allowed")
	}

	if ip := net.ParseIP(normalizedHost); ip != nil {
		if !isPublicSubscriptionAddr(ip) {
			return fmt.Errorf("subscription host %s is not a public address", normalizedHost)
		}
		return nil
	}

	ips, err := net.LookupIP(normalizedHost)
	if err != nil {
		return fmt.Errorf("failed to resolve subscription host %s: %w", normalizedHost, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("subscription host %s resolves to no address", normalizedHost)
	}
	for _, ip := range ips {
		if !isPublicSubscriptionAddr(ip) {
			return fmt.Errorf("subscription host %s resolves to non-public address %s", normalizedHost, ip.String())
		}
	}

	return nil
}

func validateSubscriptionURL(parsedURL *url.URL) error {
	if parsedURL == nil {
		return fmt.Errorf("subscription URL is nil")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http/https allowed)", parsedURL.Scheme)
	}
	return validateSubscriptionHost(parsedURL.Hostname())
}

func FetchSubscription(subURL string, userAgent ...string) ([]ProxyNode, error) {
	// Validate URL format
	parsedURL, err := url.Parse(subURL)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription URL: %w", err)
	}

	// Only allow http and https protocols
	if err := validateSubscriptionURL(parsedURL); err != nil {
		return nil, err
	}

	// Create HTTP client, skip SSL verification, set timeout
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: allowInsecureSubscriptionTLS(),
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects while fetching subscription")
			}
			return validateSubscriptionURL(req.URL)
		},
	}

	// Create request and set User-Agent
	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	ua := PredefinedUserAgents["default"]
	if len(userAgent) > 0 && userAgent[0] != "" {
		ua = ResolveUserAgent(userAgent[0])
	}
	req.Header.Set("User-Agent", ua)

	// Fetch subscription content
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription returned status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription: %w", err)
	}

	// Detect if it's Clash YAML format
	content := strings.TrimSpace(string(body))
	if isClashYAML(content) {
		return parseClashYAML(body)
	}

	// Base64 decode (auto handle padding)
	decoded, err := decodeBase64(content)
	if err != nil {
		// If base64 decode fails, content might be plain text
		decoded = body
	}

	return parseProxyLines(string(decoded))
}

// parseProxyLines parses proxy link lines (vmess://, vless://, etc.)
func parseProxyLines(content string) ([]ProxyNode, error) {
	lines := strings.Split(content, "\n")
	var nodes []ProxyNode

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse nodes of different protocols
		if strings.HasPrefix(line, "vmess://") {
			node, err := parseVMessNode(line)
			if err == nil {
				nodes = append(nodes, node)
			}
		} else if strings.HasPrefix(line, "vless://") {
			node, err := parseVLESSNode(line)
			if err == nil {
				nodes = append(nodes, node)
			}
		} else if strings.HasPrefix(line, "trojan://") {
			node, err := parseTrojanNode(line)
			if err == nil {
				nodes = append(nodes, node)
			}
		} else if strings.HasPrefix(line, "ss://") {
			node, err := parseShadowsocksNode(line)
			if err == nil {
				nodes = append(nodes, node)
			}
		}
	}

	return nodes, nil
}

// isClashYAML detects if content is Clash YAML format
func isClashYAML(content string) bool {
	if !strings.Contains(content, "proxies:") {
		return false
	}

	// Standard full Clash configs have proxy-groups or rules
	if strings.Contains(content, "proxy-groups:") || strings.Contains(content, "rules:") {
		return true
	}

	// Clash/Mihomo-specific top-level keys commonly found in proxy-list-only configs
	if strings.Contains(content, "mixed-port:") || strings.Contains(content, "allow-lan:") {
		return true
	}

	// Final check: try actual YAML parse to confirm it's a valid Clash proxy list
	var config struct {
		Proxies []interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(content), &config); err == nil && len(config.Proxies) > 0 {
		return true
	}

	return false
}

// ClashConfig Clash YAML config structure
type ClashConfig struct {
	Proxies []ClashProxy `yaml:"proxies"`
}

// ClashProxy Clash proxy node
type ClashProxy struct {
	Name           string `yaml:"name"`
	Type           string `yaml:"type"`
	Server         string `yaml:"server"`
	Port           int    `yaml:"port"`
	Password       string `yaml:"password"`
	UUID           string `yaml:"uuid"`
	AlterID        int    `yaml:"alterId"`
	Cipher         string `yaml:"cipher"`
	UDP            bool   `yaml:"udp"`
	SNI            string `yaml:"sni"`
	SkipCertVerify bool   `yaml:"skip-cert-verify"`
	TLS            bool   `yaml:"tls"`
	Network        string `yaml:"network"`
	// WS config
	WSOpts *ClashWSOptions `yaml:"ws-opts"`
	// GRPC config
	GRPCOpts *ClashGRPCOptions `yaml:"grpc-opts"`
	// Flow (VLESS)
	Flow string `yaml:"flow"`
	// Reality
	RealityOpts *ClashRealityOptions `yaml:"reality-opts"`
	// Servername (Clash Meta format, used for Reality SNI)
	Servername string `yaml:"servername"`
	// Client Fingerprint
	ClientFingerprint string `yaml:"client-fingerprint"`
	// SS plugin
	Plugin     string                 `yaml:"plugin"`
	PluginOpts map[string]interface{} `yaml:"plugin-opts"`
}

// ClashWSOptions WebSocket options
type ClashWSOptions struct {
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
}

// ClashGRPCOptions gRPC options
type ClashGRPCOptions struct {
	GRPCServiceName string `yaml:"grpc-service-name"`
}

// ClashRealityOptions Reality options
type ClashRealityOptions struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id"`
}

// parseClashYAML parses Clash YAML format subscription
func parseClashYAML(data []byte) ([]ProxyNode, error) {
	var config ClashConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Clash YAML: %w", err)
	}

	var nodes []ProxyNode
	for _, proxy := range config.Proxies {
		node, err := convertClashProxy(proxy)
		if err != nil {
			continue // skip unsupported nodes
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// convertClashProxy converts a Clash proxy node to sing-box ProxyNode
func convertClashProxy(proxy ClashProxy) (ProxyNode, error) {
	switch proxy.Type {
	case "anytls":
		return convertClashAnyTLS(proxy)
	case "vmess":
		return convertClashVMess(proxy)
	case "vless":
		return convertClashVLESS(proxy)
	case "trojan":
		return convertClashTrojan(proxy)
	case "ss", "shadowsocks":
		return convertClashShadowsocks(proxy)
	default:
		return ProxyNode{}, fmt.Errorf("unsupported Clash proxy type: %s", proxy.Type)
	}
}

// convertClashAnyTLS converts an AnyTLS node
func convertClashAnyTLS(proxy ClashProxy) (ProxyNode, error) {
	node := ProxyNode{
		Name:     proxy.Name,
		Protocol: "anytls",
		Address:  proxy.Server,
		Port:     proxy.Port,
		Settings: map[string]interface{}{
			"password": proxy.Password,
		},
	}

	node.Outbound = map[string]interface{}{
		"type":        "anytls",
		"tag":         SanitizeTag("anytls", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"password":    proxy.Password,
	}

	// AnyTLS requires TLS
	tlsConfig := map[string]interface{}{
		"enabled": true,
	}
	if proxy.SNI != "" {
		tlsConfig["server_name"] = proxy.SNI
	} else {
		tlsConfig["server_name"] = proxy.Server
	}
	if proxy.SkipCertVerify {
		tlsConfig["insecure"] = true
	}
	node.Outbound["tls"] = tlsConfig

	return node, nil
}

// convertClashVMess converts a VMess node
func convertClashVMess(proxy ClashProxy) (ProxyNode, error) {
	node := ProxyNode{
		Name:     proxy.Name,
		Protocol: "vmess",
		Address:  proxy.Server,
		Port:     proxy.Port,
		Settings: map[string]interface{}{
			"id":       proxy.UUID,
			"alterId":  proxy.AlterID,
			"security": "auto",
		},
	}

	node.Outbound = map[string]interface{}{
		"type":        "vmess",
		"tag":         SanitizeTag("vmess", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"uuid":        proxy.UUID,
		"security":    "auto",
		"alter_id":    proxy.AlterID,
	}

	// Transport layer
	addClashTransport(proxy, node.Outbound)
	// TLS
	addClashTLS(proxy, node.Outbound)

	return node, nil
}

// convertClashVLESS converts a VLESS node
func convertClashVLESS(proxy ClashProxy) (ProxyNode, error) {
	node := ProxyNode{
		Name:     proxy.Name,
		Protocol: "vless",
		Address:  proxy.Server,
		Port:     proxy.Port,
		Settings: map[string]interface{}{
			"id":   proxy.UUID,
			"flow": proxy.Flow,
		},
	}

	node.Outbound = map[string]interface{}{
		"type":        "vless",
		"tag":         SanitizeTag("vless", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"uuid":        proxy.UUID,
	}

	if proxy.Flow != "" {
		node.Outbound["flow"] = proxy.Flow
	}

	// Transport layer
	addClashTransport(proxy, node.Outbound)
	// TLS (including Reality)
	addClashTLS(proxy, node.Outbound)

	return node, nil
}

// convertClashTrojan converts a Trojan node
func convertClashTrojan(proxy ClashProxy) (ProxyNode, error) {
	node := ProxyNode{
		Name:     proxy.Name,
		Protocol: "trojan",
		Address:  proxy.Server,
		Port:     proxy.Port,
		Settings: map[string]interface{}{
			"password": proxy.Password,
		},
	}

	node.Outbound = map[string]interface{}{
		"type":        "trojan",
		"tag":         SanitizeTag("trojan", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"password":    proxy.Password,
	}

	// Transport layer
	addClashTransport(proxy, node.Outbound)

	// Trojan enables TLS by default
	tlsConfig := map[string]interface{}{
		"enabled": true,
	}
	if proxy.SNI != "" {
		tlsConfig["server_name"] = proxy.SNI
	}
	if proxy.SkipCertVerify {
		tlsConfig["insecure"] = true
	}
	fp := proxy.ClientFingerprint
	if fp == "" {
		fp = "chrome"
	}
	tlsConfig["utls"] = map[string]interface{}{
		"enabled":     true,
		"fingerprint": fp,
	}
	node.Outbound["tls"] = tlsConfig

	return node, nil
}

// convertClashShadowsocks converts a Shadowsocks node
func convertClashShadowsocks(proxy ClashProxy) (ProxyNode, error) {
	node := ProxyNode{
		Name:     proxy.Name,
		Protocol: "shadowsocks",
		Address:  proxy.Server,
		Port:     proxy.Port,
		Settings: map[string]interface{}{
			"method":   proxy.Cipher,
			"password": proxy.Password,
		},
	}

	node.Outbound = map[string]interface{}{
		"type":        "shadowsocks",
		"tag":         SanitizeTag("ss", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"method":      proxy.Cipher,
		"password":    proxy.Password,
	}

	return node, nil
}

// addClashTransport adds Clash node transport config to sing-box outbound
func addClashTransport(proxy ClashProxy, outbound map[string]interface{}) {
	if proxy.Network == "" || proxy.Network == "tcp" {
		return
	}

	transport := map[string]interface{}{
		"type": proxy.Network,
	}

	if proxy.Network == "ws" && proxy.WSOpts != nil {
		if proxy.WSOpts.Path != "" {
			transport["path"] = proxy.WSOpts.Path
		}
		if host, ok := proxy.WSOpts.Headers["Host"]; ok && host != "" {
			transport["headers"] = map[string]interface{}{
				"Host": host,
			}
		}
	}

	if proxy.Network == "grpc" && proxy.GRPCOpts != nil {
		if proxy.GRPCOpts.GRPCServiceName != "" {
			transport["service_name"] = proxy.GRPCOpts.GRPCServiceName
		}
	}

	outbound["transport"] = transport
}

// addClashTLS adds Clash node TLS config to sing-box outbound
func addClashTLS(proxy ClashProxy, outbound map[string]interface{}) {
	if !proxy.TLS && proxy.RealityOpts == nil {
		return
	}

	tlsConfig := map[string]interface{}{
		"enabled": true,
	}
	sni := proxy.SNI
	if sni == "" {
		sni = proxy.Servername
	}
	if sni != "" {
		tlsConfig["server_name"] = sni
	}
	if proxy.SkipCertVerify {
		tlsConfig["insecure"] = true
	}

	// Reality
	if proxy.RealityOpts != nil {
		tlsConfig["reality"] = map[string]interface{}{
			"enabled":    true,
			"public_key": proxy.RealityOpts.PublicKey,
			"short_id":   proxy.RealityOpts.ShortID,
		}
	}

	// uTLS
	fp := proxy.ClientFingerprint
	if fp == "" {
		fp = "chrome"
	}
	tlsConfig["utls"] = map[string]interface{}{
		"enabled":     true,
		"fingerprint": fp,
	}

	outbound["tls"] = tlsConfig
}

// parseVMessNode parses a VMess node
func parseVMessNode(link string) (ProxyNode, error) {
	// Remove vmess:// prefix
	link = strings.TrimPrefix(link, "vmess://")

	// Base64 decode (auto handle padding)
	decoded, err := decodeBase64(link)
	if err != nil {
		return ProxyNode{}, err
	}

	// Parse JSON
	var vmess VMess
	if err := json.Unmarshal(decoded, &vmess); err != nil {
		return ProxyNode{}, err
	}

	// Convert to unified format
	node := ProxyNode{
		Name:     vmess.PS,
		Protocol: "vmess",
		Address:  vmess.Add,
		Settings: map[string]interface{}{
			"id":       vmess.ID,
			"alterId":  vmess.Aid,
			"security": "auto",
		},
	}

	// Parse port
	_, _ = fmt.Sscanf(vmess.Port, "%d", &node.Port)

	// Build sing-box outbound config
	// Use unique tag to avoid conflicts during load balancing
	node.Outbound = map[string]interface{}{
		"type":        "vmess",
		"tag":         SanitizeTag("vmess", vmess.Add, node.Port),
		"server":      vmess.Add,
		"server_port": node.Port,
		"uuid":        vmess.ID,
		"security":    "auto",
		"alter_id":    parseIntOrZero(vmess.Aid),
	}

	// Add transport layer config
	if vmess.Net != "" && vmess.Net != "tcp" {
		transport := map[string]interface{}{
			"type": vmess.Net,
		}

		if vmess.Net == "ws" {
			if vmess.Path != "" {
				transport["path"] = vmess.Path
			}
			if vmess.Host != "" {
				transport["headers"] = map[string]interface{}{
					"Host": vmess.Host,
				}
			}
		}

		node.Outbound["transport"] = transport
	}

	// Add TLS config
	if vmess.TLS == "tls" {
		tlsConfig := map[string]interface{}{
			"enabled": true,
		}
		if vmess.SNI != "" {
			tlsConfig["server_name"] = vmess.SNI
		} else if vmess.Host != "" {
			tlsConfig["server_name"] = vmess.Host
		}
		node.Outbound["tls"] = tlsConfig
	}

	return node, nil
}

// parseVLESSNode parses a VLESS node
func parseVLESSNode(link string) (ProxyNode, error) {
	// Remove vless:// prefix
	link = strings.TrimPrefix(link, "vless://")

	// Parse URL
	parts := strings.SplitN(link, "@", 2)
	if len(parts) != 2 {
		return ProxyNode{}, fmt.Errorf("invalid vless link")
	}

	uuid := parts[0]
	rest := parts[1]

	// Parse address and port
	addressParts := strings.SplitN(rest, "?", 2)
	addressPort := addressParts[0]

	addrPort := strings.SplitN(addressPort, ":", 2)
	if len(addrPort) != 2 {
		return ProxyNode{}, fmt.Errorf("invalid address:port")
	}

	address := addrPort[0]
	port := 0
	_, _ = fmt.Sscanf(addrPort[1], "%d", &port)

	// Parse query parameters
	query := ""
	name := ""
	if len(addressParts) > 1 {
		queryAndName := addressParts[1]
		queryParts := strings.SplitN(queryAndName, "#", 2)
		query = queryParts[0]
		if len(queryParts) > 1 {
			name, _ = url.QueryUnescape(queryParts[1])
		}
	}

	params, _ := url.ParseQuery(query)

	node := ProxyNode{
		Name:     name,
		Protocol: "vless",
		Address:  address,
		Port:     port,
		Settings: map[string]interface{}{
			"id":         uuid,
			"encryption": params.Get("encryption"),
			"flow":       params.Get("flow"),
		},
	}

	// Build sing-box outbound config
	node.Outbound = map[string]interface{}{
		"type":        "vless",
		"tag":         SanitizeTag("vless", address, port),
		"server":      address,
		"server_port": port,
		"uuid":        uuid,
	}

	if flow := params.Get("flow"); flow != "" {
		node.Outbound["flow"] = flow
	}

	// Add transport layer config
	network := params.Get("type")
	security := params.Get("security")

	if network != "" && network != "tcp" {
		transport := map[string]interface{}{
			"type": network,
		}

		if network == "ws" {
			if path := params.Get("path"); path != "" {
				transport["path"] = path
			}
			if host := params.Get("host"); host != "" {
				transport["headers"] = map[string]interface{}{
					"Host": host,
				}
			}
		}

		node.Outbound["transport"] = transport
	}

	// Add TLS config
	if security == "tls" || security == "reality" {
		tlsConfig := map[string]interface{}{
			"enabled": true,
		}
		if sni := params.Get("sni"); sni != "" {
			tlsConfig["server_name"] = sni
		}
		if security == "reality" {
			tlsConfig["reality"] = map[string]interface{}{
				"enabled":    true,
				"public_key": params.Get("pbk"),
				"short_id":   params.Get("sid"),
			}
		}
		// Add uTLS config (required for Vision flow)
		fp := params.Get("fp")
		if fp == "" {
			fp = "chrome" // default use chrome fingerprint
		}
		tlsConfig["utls"] = map[string]interface{}{
			"enabled":     true,
			"fingerprint": fp,
		}
		node.Outbound["tls"] = tlsConfig
	}

	return node, nil
}

// parseTrojanNode parses a Trojan node
func parseTrojanNode(link string) (ProxyNode, error) {
	// Remove trojan:// prefix
	link = strings.TrimPrefix(link, "trojan://")

	// Parse URL
	parts := strings.SplitN(link, "@", 2)
	if len(parts) != 2 {
		return ProxyNode{}, fmt.Errorf("invalid trojan link")
	}

	password := parts[0]
	rest := parts[1]

	// Parse address and port
	addressParts := strings.SplitN(rest, "?", 2)
	addressPort := addressParts[0]

	addrPort := strings.SplitN(addressPort, ":", 2)
	if len(addrPort) != 2 {
		return ProxyNode{}, fmt.Errorf("invalid address:port")
	}

	address := addrPort[0]
	port := 0
	_, _ = fmt.Sscanf(addrPort[1], "%d", &port)

	// Parse query parameters
	name := ""
	query := ""
	if len(addressParts) > 1 {
		queryAndName := addressParts[1]
		if idx := strings.Index(queryAndName, "#"); idx != -1 {
			query = queryAndName[:idx]
			name, _ = url.QueryUnescape(queryAndName[idx+1:])
		} else {
			query = queryAndName
		}
	}

	params, _ := url.ParseQuery(query)

	node := ProxyNode{
		Name:     name,
		Protocol: "trojan",
		Address:  address,
		Port:     port,
		Settings: map[string]interface{}{
			"password": password,
		},
	}

	// Build sing-box outbound config
	node.Outbound = map[string]interface{}{
		"type":        "trojan",
		"tag":         SanitizeTag("trojan", address, port),
		"server":      address,
		"server_port": port,
		"password":    password,
	}

	// Add transport layer config
	network := params.Get("type")
	if network != "" && network != "tcp" {
		transport := map[string]interface{}{
			"type": network,
		}

		if network == "ws" {
			if path := params.Get("path"); path != "" {
				transport["path"] = path
			}
			if host := params.Get("host"); host != "" {
				transport["headers"] = map[string]interface{}{
					"Host": host,
				}
			}
		}

		node.Outbound["transport"] = transport
	}

	// Add TLS config (Trojan enables TLS by default)
	tlsConfig := map[string]interface{}{
		"enabled": true,
	}
	if sni := params.Get("sni"); sni != "" {
		tlsConfig["server_name"] = sni
	}
	// Add uTLS config
	fp := params.Get("fp")
	if fp == "" {
		fp = "chrome" // default use chrome fingerprint
	}
	tlsConfig["utls"] = map[string]interface{}{
		"enabled":     true,
		"fingerprint": fp,
	}
	node.Outbound["tls"] = tlsConfig

	return node, nil
}

// parseShadowsocksNode parses a Shadowsocks node
// Supported formats:
// 1. SIP002: ss://BASE64(method:password)@server:port#name
// 2. SS2022 multi-user: ss://BASE64(method:serverKey:userKey)@server:port#name
// 3. Legacy format: ss://BASE64(method:password@server:port)#name
func parseShadowsocksNode(link string) (ProxyNode, error) {
	// Remove ss:// prefix
	link = strings.TrimPrefix(link, "ss://")

	// Split link and comment (if there is a # symbol)
	var name string
	linkParts := strings.SplitN(link, "#", 2)
	if len(linkParts) == 2 {
		link = linkParts[0]
		name, _ = url.QueryUnescape(linkParts[1])
	}

	// Try to parse SIP002 format: method:password@server:port
	// or Base64(method:password)@server:port
	// or SS2022: Base64(method:serverKey:userKey)@server:port
	var method, password, address string
	var port int

	// Check if it contains @
	if strings.Contains(link, "@") {
		// Could be SIP002 format or userinfo@host format
		parts := strings.SplitN(link, "@", 2)

		// Try to base64 decode the first part (method:password or method:serverKey:userKey)
		userInfo, err := decodeBase64(parts[0])
		if err != nil {
			// Not base64, use directly
			userInfo = []byte(parts[0])
		}

		// Parse method:password or method:serverKey:userKey
		// Use SplitN(..., 2) to split only at the first colon
		// This way for SS2022 format, password will be "serverKey:userKey"
		userInfoStr := string(userInfo)
		colonIdx := strings.Index(userInfoStr, ":")
		if colonIdx == -1 {
			return ProxyNode{}, fmt.Errorf("invalid method:password format")
		}
		method = userInfoStr[:colonIdx]
		password = userInfoStr[colonIdx+1:]

		// Parse server:port
		addressPort := strings.SplitN(parts[1], ":", 2)
		if len(addressPort) != 2 {
			return ProxyNode{}, fmt.Errorf("invalid address:port")
		}
		address = addressPort[0]
		_, _ = fmt.Sscanf(addressPort[1], "%d", &port)
	} else {
		// Legacy format: the entire link is Base64 encoded method:password@server:port
		decoded, err := decodeBase64(link)
		if err != nil {
			return ProxyNode{}, fmt.Errorf("failed to decode ss link: %w", err)
		}

		// Parse format: method:password@server:port
		decodedStr := string(decoded)
		parts := strings.SplitN(decodedStr, "@", 2)
		if len(parts) != 2 {
			return ProxyNode{}, fmt.Errorf("invalid ss link format")
		}

		// Parse method:password (supports SS2022 multi-user format)
		colonIdx := strings.Index(parts[0], ":")
		if colonIdx == -1 {
			return ProxyNode{}, fmt.Errorf("invalid method:password")
		}
		method = parts[0][:colonIdx]
		password = parts[0][colonIdx+1:]

		addressPort := strings.SplitN(parts[1], ":", 2)
		if len(addressPort) != 2 {
			return ProxyNode{}, fmt.Errorf("invalid address:port")
		}

		address = addressPort[0]
		_, _ = fmt.Sscanf(addressPort[1], "%d", &port)
	}

	if name == "" {
		name = fmt.Sprintf("SS-%s:%d", address, port)
	}

	node := ProxyNode{
		Name:     name,
		Protocol: "shadowsocks",
		Address:  address,
		Port:     port,
		Settings: map[string]interface{}{
			"method":   method,
			"password": password,
		},
	}

	// Build sing-box outbound config
	node.Outbound = map[string]interface{}{
		"type":        "shadowsocks",
		"tag":         SanitizeTag("ss", address, port),
		"server":      address,
		"server_port": port,
		"method":      method,
		"password":    password,
	}

	return node, nil
}

// parseIntOrZero converts string to int, returns 0 on failure
func parseIntOrZero(s string) int {
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}

// SanitizeTag generates a normalized unique node tag
// Format: {protocol}-{address}-{port}
// Removes special characters to ensure tag is valid in sing-box config
func SanitizeTag(protocol, address string, port int) string {
	// Replace unsafe characters
	safeAddress := strings.ReplaceAll(address, ".", "_")
	safeAddress = strings.ReplaceAll(safeAddress, ":", "_")
	safeAddress = strings.ReplaceAll(safeAddress, "-", "_")
	return fmt.Sprintf("%s-%s-%d", protocol, safeAddress, port)
}

// Predefined User-Agent list
var PredefinedUserAgents = map[string]string{
	"clash-verge": "clash-verge/v2.4.0",
	"clash-meta":  "ClashMeta/v1.18.0",
	"v2rayn":      "v2rayN/6.0",
	"v2rayng":     "v2rayNG/1.8.0",
	"default":     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

// SubscriptionEntry single subscription entry
type SubscriptionEntry struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	URL            string      `json:"url"`
	UserAgent      string      `json:"user_agent,omitempty"`      // User-Agent used for requests
	AutoUpdate     bool        `json:"auto_update,omitempty"`     // whether to auto update
	UpdateInterval int         `json:"update_interval,omitempty"` // auto update interval (hours), 0 means disabled
	LastUpdated    string      `json:"last_updated,omitempty"`    // last update time (RFC3339)
	Nodes          []ProxyNode `json:"nodes"`
}

// SubscriptionData multi-subscription data (compatible with old format)
type SubscriptionData struct {
	// Old format fields (for compatibility)
	URL   string      `json:"url,omitempty"`
	Nodes []ProxyNode `json:"nodes,omitempty"`
	// New format fields
	Subscriptions []SubscriptionEntry `json:"subscriptions,omitempty"`
}

// getSubscriptionFilePath gets the subscription file path
// Note: stored in data directory not singbox directory, to avoid being loaded by sing-box -C
func getSubscriptionFilePath() string {
	baseDir := os.Getenv("DATA_DIR")
	if baseDir == "" {
		// Default to current working directory
		baseDir, _ = os.Getwd()
	}
	return filepath.Join(baseDir, "subscription.json")
}

// SaveSubscriptions saves multi-subscription data to file
func SaveSubscriptions(data SubscriptionData) error {
	subscriptionFile := getSubscriptionFilePath()

	// Ensure directory exists
	dir := filepath.Dir(subscriptionFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscription data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(subscriptionFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write subscription file: %w", err)
	}

	return nil
}

// LoadSubscriptions loads multi-subscription data from file
func LoadSubscriptions() (*SubscriptionData, error) {
	subscriptionFile := getSubscriptionFilePath()

	// Check if file exists
	if _, err := os.Stat(subscriptionFile); os.IsNotExist(err) {
		return &SubscriptionData{Subscriptions: []SubscriptionEntry{}}, nil
	}

	// Read file
	data, err := os.ReadFile(subscriptionFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription file: %w", err)
	}

	// Deserialize
	var subData SubscriptionData
	if err := json.Unmarshal(data, &subData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscription data: %w", err)
	}

	// Compatible with old format: if there's old URL field but no subscriptions, migrate data
	if subData.URL != "" && len(subData.Subscriptions) == 0 {
		subData.Subscriptions = []SubscriptionEntry{
			{
				ID:    generateSubscriptionID(),
				Name:  "Default Subscription",
				URL:   subData.URL,
				Nodes: subData.Nodes,
			},
		}
		// Clear old fields
		subData.URL = ""
		subData.Nodes = nil
		// Save migrated data
		_ = SaveSubscriptions(subData)
	}

	if subData.Subscriptions == nil {
		subData.Subscriptions = []SubscriptionEntry{}
	}

	return &subData, nil
}

// generateSubscriptionID generates a subscription ID
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// AddSubscription adds a subscription
func AddSubscription(name, subURL, userAgent string) (*SubscriptionEntry, error) {
	data, err := LoadSubscriptions()
	if err != nil {
		return nil, err
	}

	// Fetch and parse subscription nodes
	nodes, err := FetchSubscription(subURL, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	entry := SubscriptionEntry{
		ID:          generateSubscriptionID(),
		Name:        name,
		URL:         subURL,
		UserAgent:   userAgent,
		LastUpdated: time.Now().Format(time.RFC3339),
		Nodes:       nodes,
	}

	data.Subscriptions = append(data.Subscriptions, entry)

	if err := SaveSubscriptions(*data); err != nil {
		return nil, err
	}

	return &entry, nil
}

// UpdateSubscription updates a subscription (refreshes nodes)
func UpdateSubscription(id string) (*SubscriptionEntry, error) {
	data, err := LoadSubscriptions()
	if err != nil {
		return nil, err
	}

	for i, sub := range data.Subscriptions {
		if sub.ID == id {
			// Re-fetch nodes (using stored User-Agent)
			nodes, err := FetchSubscription(sub.URL, sub.UserAgent)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch subscription: %w", err)
			}
			data.Subscriptions[i].Nodes = nodes
			data.Subscriptions[i].LastUpdated = time.Now().Format(time.RFC3339)

			if err := SaveSubscriptions(*data); err != nil {
				return nil, err
			}

			return &data.Subscriptions[i], nil
		}
	}

	return nil, fmt.Errorf("subscription not found: %s", id)
}

// UpdateSubscriptionSettings updates subscription auto-update settings
func UpdateSubscriptionSettings(id string, autoUpdate bool, updateInterval int) (*SubscriptionEntry, error) {
	data, err := LoadSubscriptions()
	if err != nil {
		return nil, err
	}

	for i, sub := range data.Subscriptions {
		if sub.ID == id {
			data.Subscriptions[i].AutoUpdate = autoUpdate
			data.Subscriptions[i].UpdateInterval = updateInterval
			if err := SaveSubscriptions(*data); err != nil {
				return nil, err
			}
			return &data.Subscriptions[i], nil
		}
	}

	return nil, fmt.Errorf("subscription not found: %s", id)
}

// DeleteSubscription deletes a subscription
func DeleteSubscription(id string) error {
	data, err := LoadSubscriptions()
	if err != nil {
		return err
	}

	for i, sub := range data.Subscriptions {
		if sub.ID == id {
			data.Subscriptions = append(data.Subscriptions[:i], data.Subscriptions[i+1:]...)
			return SaveSubscriptions(*data)
		}
	}

	return fmt.Errorf("subscription not found: %s", id)
}

// GetAllNodes gets all nodes from all subscriptions
func GetAllNodes() ([]ProxyNode, error) {
	data, err := LoadSubscriptions()
	if err != nil {
		return nil, err
	}

	var allNodes []ProxyNode
	for _, sub := range data.Subscriptions {
		allNodes = append(allNodes, sub.Nodes...)
	}

	return allNodes, nil
}

// RefreshAllSubscriptions refreshes all subscriptions
func RefreshAllSubscriptions() (*SubscriptionData, error) {
	data, err := LoadSubscriptions()
	if err != nil {
		return nil, err
	}

	for i, sub := range data.Subscriptions {
		nodes, err := FetchSubscription(sub.URL, sub.UserAgent)
		if err != nil {
			// Log error but continue processing other subscriptions
			continue
		}
		data.Subscriptions[i].Nodes = nodes
	}

	if err := SaveSubscriptions(*data); err != nil {
		return nil, err
	}

	return data, nil
}

// ProbeResultUpdate probe result update
type ProbeResultUpdate struct {
	Tag         string `json:"tag"`
	Latency     int64  `json:"latency"`
	Online      bool   `json:"online"`
	LastProbe   string `json:"last_probe"`
	SuccessRate int    `json:"success_rate"`
}

// UpdateProbeResults updates node probe results to subscription file
func UpdateProbeResults(results []ProbeResultUpdate) error {
	data, err := LoadSubscriptions()
	if err != nil {
		return err
	}

	// Build tag -> result mapping
	resultMap := make(map[string]ProbeResultUpdate)
	for _, r := range results {
		resultMap[r.Tag] = r
	}

	// Update probe results for nodes in each subscription
	for i := range data.Subscriptions {
		for j := range data.Subscriptions[i].Nodes {
			node := &data.Subscriptions[i].Nodes[j]
			// Get node tag
			tag := ""
			if node.Outbound != nil {
				if t, ok := node.Outbound["tag"].(string); ok {
					tag = t
				}
			}
			if tag == "" {
				tag = SanitizeTag(node.Protocol, node.Address, node.Port)
			}

			// Update probe results
			if result, exists := resultMap[tag]; exists {
				node.Latency = result.Latency
				node.Online = result.Online
				node.LastProbe = result.LastProbe
				node.SuccessRate = result.SuccessRate
			}
		}
	}

	return SaveSubscriptions(*data)
}

// SpeedTestUpdate proxy speed test result update
type SpeedTestUpdate struct {
	Tag       string  `json:"tag"`
	Latency   int64   `json:"latency"`
	SpeedKBps float64 `json:"speed_kbps"`
	Online    bool    `json:"online"`
	LastProbe string  `json:"last_probe"`
}

// UpdateSpeedTestResults writes proxy speed test results to subscription file (with speed)
func UpdateSpeedTestResults(results []SpeedTestUpdate) error {
	data, err := LoadSubscriptions()
	if err != nil {
		return err
	}
	m := make(map[string]SpeedTestUpdate, len(results))
	for _, r := range results {
		m[r.Tag] = r
	}
	for i := range data.Subscriptions {
		for j := range data.Subscriptions[i].Nodes {
			node := &data.Subscriptions[i].Nodes[j]
			tag := ""
			if node.Outbound != nil {
				if t, ok := node.Outbound["tag"].(string); ok {
					tag = t
				}
			}
			if tag == "" {
				tag = SanitizeTag(node.Protocol, node.Address, node.Port)
			}
			if r, ok := m[tag]; ok {
				node.Latency = r.Latency
				node.SpeedKBps = r.SpeedKBps
				node.Online = r.Online
				node.LastProbe = r.LastProbe
			}
		}
	}
	return SaveSubscriptions(*data)
}
