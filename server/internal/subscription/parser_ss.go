package subscription

import (
	"fmt"
	"net/url"
	"strings"

	"singbox-config-service/internal/pkg/types"
)

func parseShadowsocksNode(link string) (types.ProxyNode, error) {
	link = strings.TrimPrefix(link, "ss://")

	var name string
	linkParts := strings.SplitN(link, "#", 2)
	if len(linkParts) == 2 {
		link = linkParts[0]
		name, _ = url.QueryUnescape(linkParts[1])
	}

	var method, password, address string
	var port int

	if strings.Contains(link, "@") {
		parts := strings.SplitN(link, "@", 2)

		userInfo, err := decodeBase64(parts[0])
		if err != nil {
			userInfo = []byte(parts[0])
		}

		userInfoStr := string(userInfo)
		colonIdx := strings.Index(userInfoStr, ":")
		if colonIdx == -1 {
			return types.ProxyNode{}, fmt.Errorf("invalid method:password format")
		}
		method = userInfoStr[:colonIdx]
		password = userInfoStr[colonIdx+1:]

		addressPort := strings.SplitN(parts[1], ":", 2)
		if len(addressPort) != 2 {
			return types.ProxyNode{}, fmt.Errorf("invalid address:port")
		}
		address = addressPort[0]
		_, _ = fmt.Sscanf(addressPort[1], "%d", &port)
	} else {
		decoded, err := decodeBase64(link)
		if err != nil {
			return types.ProxyNode{}, fmt.Errorf("failed to decode ss link: %w", err)
		}

		decodedStr := string(decoded)
		parts := strings.SplitN(decodedStr, "@", 2)
		if len(parts) != 2 {
			return types.ProxyNode{}, fmt.Errorf("invalid ss link format")
		}

		colonIdx := strings.Index(parts[0], ":")
		if colonIdx == -1 {
			return types.ProxyNode{}, fmt.Errorf("invalid method:password")
		}
		method = parts[0][:colonIdx]
		password = parts[0][colonIdx+1:]

		addressPort := strings.SplitN(parts[1], ":", 2)
		if len(addressPort) != 2 {
			return types.ProxyNode{}, fmt.Errorf("invalid address:port")
		}
		address = addressPort[0]
		_, _ = fmt.Sscanf(addressPort[1], "%d", &port)
	}

	if name == "" {
		name = fmt.Sprintf("SS-%s:%d", address, port)
	}

	node := types.ProxyNode{
		Name:     name,
		Protocol: "shadowsocks",
		Address:  address,
		Port:     port,
		Settings: map[string]interface{}{
			"method":   method,
			"password": password,
		},
	}

	node.Outbound = map[string]interface{}{
		"type":        "shadowsocks",
		"tag":         types.SanitizeTag("ss", address, port),
		"server":      address,
		"server_port": port,
		"method":      method,
		"password":    password,
	}

	return node, nil
}
