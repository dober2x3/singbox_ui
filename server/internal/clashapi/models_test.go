package clashapi

import (
	"encoding/json"
	"testing"
)

func TestProxiesResponseUnmarshal(t *testing.T) {
	data := `{
		"proxies": {
			"Proxy": {
				"type": "Selector",
				"now": "NodeSG",
				"all": ["NodeSG", "NodeJP", "NodeUS"]
			}
		}
	}`
	var resp ProxiesResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatal(err)
	}
	proxy, ok := resp.Proxies["Proxy"]
	if !ok {
		t.Fatal("expected Proxy in map")
	}
	if proxy.Type != "Selector" {
		t.Fatalf("expected Selector, got %s", proxy.Type)
	}
	if len(proxy.All) != 3 {
		t.Fatalf("expected 3 items in All, got %d", len(proxy.All))
	}
}

func TestConnectionsResponseUnmarshal(t *testing.T) {
	data := `{
		"download_total": 1000,
		"upload_total": 500,
		"connections": [
			{
				"id": "abc123",
				"metadata": {
					"network": "tcp",
					"host": "example.com"
				}
			}
		]
	}`
	var resp ConnectionsResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.DownloadTotal != 1000 {
		t.Fatalf("expected 1000, got %d", resp.DownloadTotal)
	}
	if len(resp.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(resp.Connections))
	}
}

func TestDelayResponseUnmarshal(t *testing.T) {
	data := `{"NodeSG": 123}`
	var resp DelayResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Delay != 123 {
		t.Fatalf("expected 123, got %d", resp.Delay)
	}
}

func TestTrafficMessageUnmarshal(t *testing.T) {
	data := []byte{
		0, 0, 0, 0, 0, 0, 0, 100,
		0, 0, 0, 0, 0, 0, 0, 200,
	}
	var msg TrafficMessage
	if err := msg.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if msg.Up != 100 {
		t.Fatalf("expected up=100, got %d", msg.Up)
	}
	if msg.Down != 200 {
		t.Fatalf("expected down=200, got %d", msg.Down)
	}
}

func TestMemoryMessageUnmarshal(t *testing.T) {
	data := `{"inuse": 1048576, "oslimit": 4294967296}`
	var msg MemoryMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Inuse != 1048576 {
		t.Fatalf("expected inuse=1048576, got %d", msg.Inuse)
	}
	if msg.OSLimit != 4294967296 {
		t.Fatalf("expected oslimit=4294967296, got %d", msg.OSLimit)
	}
}

func TestRulesResponseUnmarshal(t *testing.T) {
	data := `{"rules": [{"type": "DOMAIN", "payload": "example.com", "proxy": "Proxy"}]}`
	var resp RulesResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
	}
	if resp.Rules[0].Type != "DOMAIN" {
		t.Fatalf("expected DOMAIN, got %s", resp.Rules[0].Type)
	}
}

func TestConfigResponseUnmarshal(t *testing.T) {
	data := `{"mode": "Rule"}`
	var resp ConfigResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Mode != "Rule" {
		t.Fatalf("expected Rule, got %s", resp.Mode)
	}
}

func TestLogEntryUnmarshal(t *testing.T) {
	data := `{"type": "warning", "payload": "connection rejected"}`
	var entry LogEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Type != "warning" {
		t.Fatalf("expected warning, got %s", entry.Type)
	}
	if entry.Payload != "connection rejected" {
		t.Fatalf("expected 'connection rejected', got %s", entry.Payload)
	}
}

func TestDelayResponseUnmarshalEmpty(t *testing.T) {
	data := `{}`
	var resp DelayResponse
	if err := json.Unmarshal([]byte(data), &resp); err == nil {
		t.Fatal("expected error for empty delay response")
	}
}

func TestTrafficMessageUnmarshalShort(t *testing.T) {
	data := []byte{0, 1, 2, 3}
	var msg TrafficMessage
	if err := msg.UnmarshalBinary(data); err == nil {
		t.Fatal("expected error for short traffic message")
	}
}
