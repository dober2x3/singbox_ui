package subscription

import (
	"testing"
)

// TestParseVMessNode_Basic verifies parsing a vmess:// URI with base64-encoded JSON for a TCP connection.
func TestParseVMessNode_Basic(t *testing.T) {
	link := "vmess://eyJ2IjoiMiIsInBzIjoiVk1lc3MtMTU0LjE3LjMuMjE4IiwiYWRkIjoiMTU0LjE3LjMuMjE4IiwicG9ydCI6IjIwMjMiLCJpZCI6IjE2Njg2Y2QzLTRhNTUtNDA5OS1iMmU3LTM5YzhjZjU3MTQwNSIsImFpZCI6IjAiLCJuZXQiOiJ0Y3AiLCJ0eXBlIjoibm9uZSIsImhvc3QiOiIiLCJwYXRoIjoiIiwidGxzIjoiIn0="
	node, err := parseVMessNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Protocol != "vmess" {
		t.Errorf("expected protocol vmess, got %s", node.Protocol)
	}
	if node.Address != "154.17.3.218" {
		t.Errorf("expected address 154.17.3.218, got %s", node.Address)
	}
	if node.Port != 2023 {
		t.Errorf("expected port 2023, got %d", node.Port)
	}
	if node.Name != "VMess-154.17.3.218" {
		t.Errorf("expected name VMess-154.17.3.218, got %s", node.Name)
	}
}

// TestParseVMessNode_WithTLSAndWSS verifies parsing a vmess:// URI with WebSocket transport and TLS.
func TestParseVMessNode_WithTLSAndWSS(t *testing.T) {
	link := "vmess://eyJ2IjoiMiIsInBzIjoiVk1lc3MtV1MtVExTIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0IjoiNDQzIiwiaWQiOiJhYmMxMjMtYWJjMTItYWJjMTItYWJjMTJhYmMxMjNhYmMiLCJhaWQiOiI2NCIsIm5ldCI6IndzIiwidHlwZSI6Im5vbmUiLCJob3N0IjoiZXhhbXBsZS5jb20iLCJwYXRoIjoiL3dzIiwidGxzIjoidGxzIiwic25pIjoiZXhhbXBsZS5jb20ifQ=="
	node, err := parseVMessNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Protocol != "vmess" {
		t.Errorf("expected protocol vmess, got %s", node.Protocol)
	}
	if node.Address != "example.com" {
		t.Errorf("expected address example.com, got %s", node.Address)
	}
	if node.Port != 443 {
		t.Errorf("expected port 443, got %d", node.Port)
	}
	if node.Name != "VMess-WS-TLS" {
		t.Errorf("expected name VMess-WS-TLS, got %s", node.Name)
	}
	if node.Outbound["transport"] == nil {
		t.Error("expected transport config")
	}
	if node.Outbound["tls"] == nil {
		t.Error("expected tls config")
	}
}

// TestParseVMessNode_Invalid verifies that a vmess:// URI with invalid base64 returns an error.
func TestParseVMessNode_Invalid(t *testing.T) {
	_, err := parseVMessNode("vmess://invalid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}
