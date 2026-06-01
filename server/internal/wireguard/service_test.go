package wireguard

import (
	"testing"
)

// TestGeneratePrivateKey tests that GeneratePrivateKey produces a valid base64-encoded key.
func TestGeneratePrivateKey(t *testing.T) {
	svc := NewService(t.TempDir())
	key, err := svc.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("GeneratePrivateKey() error = %v", err)
	}
	if len(key) != 44 { // base64 encoded 32 bytes
		t.Errorf("key length = %d, want 44", len(key))
	}
}

// TestGeneratePublicKey tests that GeneratePublicKey correctly derives a public key from a private key.
func TestGeneratePublicKey(t *testing.T) {
	svc := NewService(t.TempDir())
	priv, _ := svc.GeneratePrivateKey()
	pub, err := svc.GeneratePublicKey(priv)
	if err != nil {
		t.Fatalf("GeneratePublicKey() error = %v", err)
	}
	if len(pub) != 44 {
		t.Errorf("public key length = %d, want 44", len(pub))
	}
}

// TestGeneratePublicKey_invalidBase64 tests that GeneratePublicKey returns an error for invalid base64 input.
func TestGeneratePublicKey_invalidBase64(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GeneratePublicKey("invalid-base64!")
	if err == nil {
		t.Error("GeneratePublicKey() expected error for invalid base64")
	}
}

// TestGenerateWireGuardKeysWithCache tests that keys are generated on first call and cached on subsequent calls.
func TestGenerateWireGuardKeysWithCache(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// First call should generate new keys
	resp, err := svc.GenerateWireGuardKeysWithCache("10.0.0.1")
	if err != nil {
		t.Fatalf("GenerateWireGuardKeysWithCache() error = %v", err)
	}
	if resp.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want %q", resp.IP, "10.0.0.1")
	}
	if resp.PrivateKey == "" || resp.PublicKey == "" {
		t.Error("PrivateKey or PublicKey is empty")
	}

	// Second call with same IP should return cached keys
	resp2, err := svc.GenerateWireGuardKeysWithCache("10.0.0.1")
	if err != nil {
		t.Fatalf("GenerateWireGuardKeysWithCache() error = %v", err)
	}
	if resp2.PrivateKey != resp.PrivateKey {
		t.Error("Second call returned different private key (cache miss)")
	}
	if resp2.PublicKey != resp.PublicKey {
		t.Error("Second call returned different public key (cache miss)")
	}
}

// TestGenerateWireGuardKeysWithCache_differentIP tests that different IPs get different keys.
func TestGenerateWireGuardKeysWithCache_differentIP(t *testing.T) {
	svc := NewService(t.TempDir())
	resp1, _ := svc.GenerateWireGuardKeysWithCache("10.0.0.1")
	resp2, _ := svc.GenerateWireGuardKeysWithCache("10.0.0.2")
	if resp1.PrivateKey == resp2.PrivateKey {
		t.Error("Different IPs should have different keys")
	}
}

// TestGenerateWireGuardKeysWithCache_noIP tests that the function returns an error when no IP is provided.
func TestGenerateWireGuardKeysWithCache_noIP(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GenerateWireGuardKeysWithCache("")
	if err == nil {
		t.Error("GenerateWireGuardKeysWithCache() expected error for empty IP")
	}
}

// TestGetKeysCache_empty tests that GetKeysCache returns an empty slice when no keys are cached.
func TestGetKeysCache_empty(t *testing.T) {
	svc := NewService(t.TempDir())
	cache, err := svc.GetKeysCache()
	if err != nil {
		t.Fatalf("GetKeysCache() error = %v", err)
	}
	if len(cache) != 0 {
		t.Errorf("cache length = %d, want 0", len(cache))
	}
}

// TestSaveAndListClientConfigFiles tests saving and listing client config files.
func TestSaveAndListClientConfigFiles(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	err := svc.SaveClientConfigFile(0, "[Interface]\nPrivateKey = test\n")
	if err != nil {
		t.Fatalf("SaveClientConfigFile() error = %v", err)
	}

	files, err := svc.ListClientConfigFiles()
	if err != nil {
		t.Fatalf("ListClientConfigFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Errorf("files count = %d, want 1", len(files))
	}
	if files[0].Name != "client0.conf" {
		t.Errorf("file name = %q, want %q", files[0].Name, "client0.conf")
	}
}

// TestSaveClientConfig tests saving and reading back a client JSON configuration.
func TestSaveClientConfig(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	err := svc.SaveClientConfig([]byte(`{"key": "value"}`))
	if err != nil {
		t.Fatalf("SaveClientConfig() error = %v", err)
	}

	data, err := svc.GetClientConfig()
	if err != nil {
		t.Fatalf("GetClientConfig() error = %v", err)
	}
	if string(data) != `{"key": "value"}` {
		t.Errorf("config data = %q, want %q", string(data), `{"key": "value"}`)
	}
}

// TestGetClientConfig_notFound tests that GetClientConfig returns an error when the config file is missing.
func TestGetClientConfig_notFound(t *testing.T) {
	svc := NewService(t.TempDir())
	_, err := svc.GetClientConfig()
	if err == nil {
		t.Error("GetClientConfig() expected error for missing file")
	}
}
