package subscription

import (
	"testing"
)

func TestParseVLESSNode_Basic(t *testing.T) {
	link := "vless://abc123-abc12-abc12-abc12-abc123abc@154.17.3.218:2023?type=tcp&security=none#VLESS-154.17.3.218"
	node, err := parseVLESSNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Protocol != "vless" {
		t.Errorf("expected protocol vless, got %s", node.Protocol)
	}
	if node.Address != "154.17.3.218" {
		t.Errorf("expected address 154.17.3.218, got %s", node.Address)
	}
	if node.Port != 2023 {
		t.Errorf("expected port 2023, got %d", node.Port)
	}
	if node.Name != "VLESS-154.17.3.218" {
		t.Errorf("expected name VLESS-154.17.3.218, got %s", node.Name)
	}
}

func TestParseVLESSNode_WithReality(t *testing.T) {
	link := "vless://abc123-abc12-abc12-abc12-abc123abc@example.com:443?type=tcp&security=reality&pbk=testPublicKey&sid=123456&fp=firefox&sni=example.com&flow=xtls-rprx-vision#VLESS-Reality"
	node, err := parseVLESSNode(link)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Protocol != "vless" {
		t.Errorf("expected protocol vless, got %s", node.Protocol)
	}
	if node.Name != "VLESS-Reality" {
		t.Errorf("expected name VLESS-Reality, got %s", node.Name)
	}

	tls, ok := node.Outbound["tls"].(map[string]interface{})
	if !ok {
		t.Fatal("expected tls config in outbound")
	}
	if tls["enabled"] != true {
		t.Error("expected tls enabled")
	}
	reality, ok := tls["reality"].(map[string]interface{})
	if !ok {
		t.Fatal("expected reality config in tls")
	}
	if reality["public_key"] != "testPublicKey" {
		t.Errorf("expected public_key testPublicKey, got %v", reality["public_key"])
	}
	if node.Outbound["flow"] != "xtls-rprx-vision" {
		t.Errorf("expected flow xtls-rprx-vision, got %v", node.Outbound["flow"])
	}
}

func TestParseVLESSNode_Invalid(t *testing.T) {
	_, err := parseVLESSNode("vless://no-at-sign")
	if err == nil {
		t.Error("expected error for invalid vless link")
	}
}
