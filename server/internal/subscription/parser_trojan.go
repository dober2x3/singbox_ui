package subscription

import (
	"fmt"
	"net/url"
	"strings"

	"singbox-config-service/internal/pkg/types"
)

func parseTrojanNode(link string) (types.ProxyNode, error) {
	link = strings.TrimPrefix(link, "trojan://")
	parts := strings.SplitN(link, "@", 2)
	if len(parts) != 2 {
		return types.ProxyNode{}, fmt.Errorf("invalid trojan link")
	}

	password := parts[0]
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

	node := types.ProxyNode{
		Name:     name,
		Protocol: "trojan",
		Address:  address,
		Port:     port,
		Settings: map[string]interface{}{
			"password": password,
		},
	}

	node.Outbound = map[string]interface{}{
		"type":        "trojan",
		"tag":         types.SanitizeTag("trojan", address, port),
		"server":      address,
		"server_port": port,
		"password":    password,
	}

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

	tlsConfig := map[string]interface{}{
		"enabled": true,
	}
	if sni := params.Get("sni"); sni != "" {
		tlsConfig["server_name"] = sni
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

	return node, nil
}
