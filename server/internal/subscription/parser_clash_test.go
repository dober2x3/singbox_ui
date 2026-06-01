package subscription

import (
	"testing"
)

// TestIsClashYAML_True verifies that valid Clash YAML with proxies is detected as Clash format.
func TestIsClashYAML_True(t *testing.T) {
	content := `
proxies:
  - name: "JP-01"
    type: vmess
    server: example.com
    port: 443
    uuid: abc-123
    alterId: 0
    cipher: auto
    tls: true
    network: ws
`
	if !isClashYAML(content) {
		t.Error("expected isClashYAML to return true for valid Clash YAML with proxies")
	}
}

// TestIsClashYAML_TrueWithProxyGroups verifies that Clash YAML with proxy-groups is detected.
func TestIsClashYAML_TrueWithProxyGroups(t *testing.T) {
	content := `
proxies:
  - name: "JP-01"
    type: vmess
    server: example.com
    port: 443

proxy-groups:
  - name: "Proxy"
    type: select
    proxies:
      - "JP-01"
`
	if !isClashYAML(content) {
		t.Error("expected isClashYAML to return true for Clash config with proxy-groups")
	}
}

// TestIsClashYAML_False verifies that non-Clash content is not detected as Clash format.
func TestIsClashYAML_False(t *testing.T) {
	content := `just some random text`
	if isClashYAML(content) {
		t.Error("expected isClashYAML to return false for non-Clash content")
	}
}

// TestParseClashYAML_VMess verifies that a Clash YAML with a vmess proxy is parsed correctly.
func TestParseClashYAML_VMess(t *testing.T) {
	data := []byte(`
proxies:
  - name: "JP-01"
    type: vmess
    server: example.com
    port: 443
    uuid: abc-123
    alterId: 0
    cipher: auto
`)
	nodes, err := parseClashYAML(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Protocol != "vmess" {
		t.Errorf("expected protocol vmess, got %s", nodes[0].Protocol)
	}
	if nodes[0].Address != "example.com" {
		t.Errorf("expected address example.com, got %s", nodes[0].Address)
	}
}

// TestParseClashYAML_Trojan verifies that a Clash YAML with a trojan proxy is parsed correctly.
func TestParseClashYAML_Trojan(t *testing.T) {
	data := []byte(`
proxies:
  - name: "US-01"
    type: trojan
    server: trojan.example.com
    port: 443
    password: secret
    sni: trojan.example.com
`)
	nodes, err := parseClashYAML(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Protocol != "trojan" {
		t.Errorf("expected protocol trojan, got %s", nodes[0].Protocol)
	}
}

// TestParseClashYAML_MultipleNodes verifies that a Clash YAML with multiple proxies of different types is parsed.
func TestParseClashYAML_MultipleNodes(t *testing.T) {
	data := []byte(`
proxies:
  - name: "JP-01"
    type: vmess
    server: jp.example.com
    port: 443
    uuid: abc-123
    alterId: 0
  - name: "US-01"
    type: trojan
    server: us.example.com
    port: 443
    password: pass
  - name: "SG-01"
    type: ss
    server: sg.example.com
    port: 8443
    cipher: aes-256-gcm
    password: ss-pass
`)
	nodes, err := parseClashYAML(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
}
