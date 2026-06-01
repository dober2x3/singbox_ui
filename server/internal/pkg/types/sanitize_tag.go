package types

import (
	"fmt"
	"strings"
)

// PredefinedUserAgents maps client names to their HTTP User-Agent strings.
var PredefinedUserAgents = map[string]string{
	"clash-verge": "clash-verge/v2.4.0",
	"clash-meta":  "ClashMeta/v1.18.0",
	"v2rayn":      "v2rayN/6.0",
	"v2rayng":     "v2rayNG/1.8.0",
	"default":     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

// SanitizeTag generates a safe tag string from protocol, address, and port (dots/colons replaced with underscores).
func SanitizeTag(protocol, address string, port int) string {
	safeAddress := strings.ReplaceAll(address, ".", "_")
	safeAddress = strings.ReplaceAll(safeAddress, ":", "_")
	safeAddress = strings.ReplaceAll(safeAddress, "-", "_")
	return fmt.Sprintf("%s-%s-%d", protocol, safeAddress, port)
}

// ResolveUserAgent returns the User-Agent string for a given client name or custom value.
func ResolveUserAgent(ua string) string {
	if ua == "" {
		return PredefinedUserAgents["default"]
	}
	if predefined, ok := PredefinedUserAgents[ua]; ok {
		return predefined
	}
	return ua
}
