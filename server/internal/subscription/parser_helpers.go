package subscription

import (
	"encoding/base64"
	"fmt"
	"strings"

	"singbox-config-service/internal/pkg/types"
)

func decodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	decoded, err := base64.URLEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}
	decoded, err = base64.StdEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}
	s = strings.TrimRight(s, "=")
	decoded, err = base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		return decoded, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}

func parseIntOrZero(s string) int {
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}

func parseProxyLines(content string) ([]types.ProxyNode, error) {
	lines := strings.Split(content, "\n")
	var nodes []types.ProxyNode
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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
