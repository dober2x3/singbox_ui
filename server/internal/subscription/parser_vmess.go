package subscription

import (
	"encoding/json"
	"fmt"
	"strings"

	"singbox-config-service/internal/pkg/types"
)

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

func parseVMessNode(link string) (types.ProxyNode, error) {
	link = strings.TrimPrefix(link, "vmess://")
	decoded, err := decodeBase64(link)
	if err != nil {
		return types.ProxyNode{}, err
	}
	var vmess VMess
	if err := json.Unmarshal(decoded, &vmess); err != nil {
		return types.ProxyNode{}, err
	}
	node := types.ProxyNode{
		Name:     vmess.PS,
		Protocol: "vmess",
		Address:  vmess.Add,
		Settings: map[string]interface{}{
			"id":       vmess.ID,
			"alterId":  vmess.Aid,
			"security": "auto",
		},
	}
	_, _ = fmt.Sscanf(vmess.Port, "%d", &node.Port)

	node.Outbound = map[string]interface{}{
		"type":        "vmess",
		"tag":         types.SanitizeTag("vmess", vmess.Add, node.Port),
		"server":      vmess.Add,
		"server_port": node.Port,
		"uuid":        vmess.ID,
		"security":    "auto",
		"alter_id":    parseIntOrZero(vmess.Aid),
	}

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
