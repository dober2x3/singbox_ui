package subscription

import (
	"testing"
)

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

func TestIsClashYAML_False(t *testing.T) {
	content := `just some random text`
	if isClashYAML(content) {
		t.Error("expected isClashYAML to return false for non-Clash content")
	}
}

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
