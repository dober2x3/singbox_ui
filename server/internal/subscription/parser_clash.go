package subscription

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"singbox-config-service/internal/pkg/types"
)

type ClashConfig struct {
	Proxies []ClashProxy `yaml:"proxies"`
}

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
	WSOpts         *ClashWSOptions        `yaml:"ws-opts"`
	GRPCOpts       *ClashGRPCOptions      `yaml:"grpc-opts"`
	Flow           string                 `yaml:"flow"`
	RealityOpts    *ClashRealityOptions   `yaml:"reality-opts"`
	Servername     string                 `yaml:"servername"`
	ClientFingerprint string              `yaml:"client-fingerprint"`
	Plugin         string                 `yaml:"plugin"`
	PluginOpts     map[string]interface{} `yaml:"plugin-opts"`
}

type ClashWSOptions struct {
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
}

type ClashGRPCOptions struct {
	GRPCServiceName string `yaml:"grpc-service-name"`
}

type ClashRealityOptions struct {
	PublicKey string `yaml:"public-key"`
	ShortID   string `yaml:"short-id"`
}

func isClashYAML(content string) bool {
	if !strings.Contains(content, "proxies:") {
		return false
	}
	if strings.Contains(content, "proxy-groups:") || strings.Contains(content, "rules:") {
		return true
	}
	if strings.Contains(content, "mixed-port:") || strings.Contains(content, "allow-lan:") {
		return true
	}
	var config struct {
		Proxies []interface{} `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(content), &config); err == nil && len(config.Proxies) > 0 {
		return true
	}
	return false
}

func parseClashYAML(data []byte) ([]types.ProxyNode, error) {
	var config ClashConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Clash YAML: %w", err)
	}
	var nodes []types.ProxyNode
	for _, proxy := range config.Proxies {
		node, err := convertClashProxy(proxy)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func convertClashProxy(proxy ClashProxy) (types.ProxyNode, error) {
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
		return types.ProxyNode{}, fmt.Errorf("unsupported Clash proxy type: %s", proxy.Type)
	}
}

func convertClashAnyTLS(proxy ClashProxy) (types.ProxyNode, error) {
	node := types.ProxyNode{
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
		"tag":         types.SanitizeTag("anytls", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"password":    proxy.Password,
	}
	tlsConfig := map[string]interface{}{"enabled": true}
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

func convertClashVMess(proxy ClashProxy) (types.ProxyNode, error) {
	node := types.ProxyNode{
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
		"tag":         types.SanitizeTag("vmess", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"uuid":        proxy.UUID,
		"security":    "auto",
		"alter_id":    proxy.AlterID,
	}
	addClashTransport(proxy, node.Outbound)
	addClashTLS(proxy, node.Outbound)
	return node, nil
}

func convertClashVLESS(proxy ClashProxy) (types.ProxyNode, error) {
	node := types.ProxyNode{
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
		"tag":         types.SanitizeTag("vless", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"uuid":        proxy.UUID,
	}
	if proxy.Flow != "" {
		node.Outbound["flow"] = proxy.Flow
	}
	addClashTransport(proxy, node.Outbound)
	addClashTLS(proxy, node.Outbound)
	return node, nil
}

func convertClashTrojan(proxy ClashProxy) (types.ProxyNode, error) {
	node := types.ProxyNode{
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
		"tag":         types.SanitizeTag("trojan", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"password":    proxy.Password,
	}
	addClashTransport(proxy, node.Outbound)
	tlsConfig := map[string]interface{}{"enabled": true}
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

func convertClashShadowsocks(proxy ClashProxy) (types.ProxyNode, error) {
	node := types.ProxyNode{
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
		"tag":         types.SanitizeTag("ss", proxy.Server, proxy.Port),
		"server":      proxy.Server,
		"server_port": proxy.Port,
		"method":      proxy.Cipher,
		"password":    proxy.Password,
	}
	return node, nil
}

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
	if proxy.RealityOpts != nil {
		tlsConfig["reality"] = map[string]interface{}{
			"enabled":    true,
			"public_key": proxy.RealityOpts.PublicKey,
			"short_id":   proxy.RealityOpts.ShortID,
		}
	}
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
