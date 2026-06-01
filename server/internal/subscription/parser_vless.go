package subscription

import (
	"fmt"
	"net/url"
	"strings"

	"singbox-config-service/internal/pkg/types"
)

// parseVLESSNode parses a vless:// URI and returns a ProxyNode with UUID, server, port, and optional transport/TLS/Reality settings.
func parseVLESSNode(link string) (types.ProxyNode, error) {
	link = strings.TrimPrefix(link, "vless://")
	parts := strings.SplitN(link, "@", 2)
	if len(parts) != 2 {
		return types.ProxyNode{}, fmt.Errorf("invalid vless link")
	}

	uuid := parts[0]
	rest := parts[1]

	addressParts := strings.SplitN(rest, "?", 2)
	addressPort := addressParts[0]
	addrPort := strings.SplitN(addressPort, ":", 2)
	if len(addrPort) != 2 {
		return types.ProxyNode{}, fmt.Errorf("invalid address:port")
	}

	address := addrPort[0]
	port := 0
	_, _ = fmt.Sscanf(addrPort[1], "%d", &port)

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

	node := types.ProxyNode{
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

	node.Outbound = map[string]interface{}{
		"type":        "vless",
		"tag":         types.SanitizeTag("vless", address, port),
		"server":      address,
		"server_port": port,
		"uuid":        uuid,
	}

	if flow := params.Get("flow"); flow != "" {
		node.Outbound["flow"] = flow
	}

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
		fp := params.Get("fp")
		if fp == "" {
			fp = "chrome"
		}
		tlsConfig["utls"] = map[string]interface{}{
			"enabled":     true,
			"fingerprint": fp,
		}
		node.Outbound["tls"] = tlsConfig
	}

	return node, nil
}
