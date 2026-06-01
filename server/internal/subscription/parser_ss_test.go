package subscription

import (
	"testing"
)

func TestParseShadowsocksNode_SIP002(t *testing.T) {
	// ss://BASE64(method:password)@server:port#name
	link := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM=@154.17.3.218:2023#SS-Example"
	node, err := parseShadowsocksNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Protocol != "shadowsocks" {
		t.Errorf("expected protocol shadowsocks, got %s", node.Protocol)
	}
	if node.Address != "154.17.3.218" {
		t.Errorf("expected address 154.17.3.218, got %s", node.Address)
	}
	if node.Port != 2023 {
		t.Errorf("expected port 2023, got %d", node.Port)
	}
	if node.Name != "SS-Example" {
		t.Errorf("expected name SS-Example, got %s", node.Name)
	}
}

func TestParseShadowsocksNode_Legacy(t *testing.T) {
	// Legacy: ss://BASE64(method:password@server:port)
	link := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmRAMTU0LjE3LjMuMjE4OjIwMjM="
	node, err := parseShadowsocksNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Protocol != "shadowsocks" {
		t.Errorf("expected protocol shadowsocks, got %s", node.Protocol)
	}
	if node.Address != "154.17.3.218" {
		t.Errorf("expected address 154.17.3.218, got %s", node.Address)
	}
	if node.Port != 2023 {
		t.Errorf("expected port 2023, got %d", node.Port)
	}
}

func TestParseShadowsocksNode_Invalid(t *testing.T) {
	_, err := parseShadowsocksNode("ss://")
	if err == nil {
		t.Error("expected error for empty ss link")
	}
}
