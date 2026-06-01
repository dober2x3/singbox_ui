package subscription

import (
	"testing"
)

// TestParseTrojanNode_Basic verifies parsing a trojan:// URI with password, server, port, and name.
func TestParseTrojanNode_Basic(t *testing.T) {
	link := "trojan://password123@154.17.3.218:2023?type=tcp#Trojan-154.17.3.218"
	node, err := parseTrojanNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Protocol != "trojan" {
		t.Errorf("expected protocol trojan, got %s", node.Protocol)
	}
	if node.Address != "154.17.3.218" {
		t.Errorf("expected address 154.17.3.218, got %s", node.Address)
	}
	if node.Port != 2023 {
		t.Errorf("expected port 2023, got %d", node.Port)
	}
	if node.Name != "Trojan-154.17.3.218" {
		t.Errorf("expected name Trojan-154.17.3.218, got %s", node.Name)
	}
	if node.Outbound["password"] != "password123" {
		t.Errorf("expected password password123, got %v", node.Outbound["password"])
	}
}

// TestParseTrojanNode_WithTLS verifies parsing a trojan:// URI with TLS sni and fingerprint settings.
func TestParseTrojanNode_WithTLS(t *testing.T) {
	link := "trojan://pass@example.com:443?type=tcp&sni=example.com&fp=chrome#Secure-Trojan"
	node, err := parseTrojanNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Name != "Secure-Trojan" {
		t.Errorf("expected name Secure-Trojan, got %s", node.Name)
	}
	tls, ok := node.Outbound["tls"].(map[string]interface{})
	if !ok {
		t.Fatal("expected tls config in outbound")
	}
	if tls["server_name"] != "example.com" {
		t.Errorf("expected server_name example.com, got %v", tls["server_name"])
	}
}

// TestParseTrojanNode_Invalid verifies that a trojan:// URI without @ returns an error.
func TestParseTrojanNode_Invalid(t *testing.T) {
	_, err := parseTrojanNode("trojan://no-at")
	if err == nil {
		t.Error("expected error for invalid trojan link")
	}
}
