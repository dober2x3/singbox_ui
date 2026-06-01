package warp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestRegisterAndBuildOutbound tests the full flow of device registration and outbound building.
func TestRegisterAndBuildOutbound(t *testing.T) {
	// Mock Cloudflare API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/v0a2158/reg" {
			_ = json.NewEncoder(w).Encode(WarpRegisterResponse{
				ID:    "test-device-id",
				Token: "test-token",
				Account: WarpAccount{
					AccountType: "free",
					WarpPlus:    false,
				},
				Config: WarpConfig{
					ClientID: "AAAA",
					Interface: WarpInterface{
						Addresses: WarpInterfaceAddr{
							V4: "172.16.0.2",
							V6: "fd01::2",
						},
					},
					Peers: []WarpPeer{
						{
							PublicKey: "bmXOC+F1FxEMF9dyiK2H5/1SUtzH0JuVo51h2wPfgyo=",
							Endpoint: WarpPeerEndpoint{
								Host: "engage.cloudflareclient.com",
							},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	warpAPIBase = server.URL

	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	rec, err := svc.RegisterDevice()
	if err != nil {
		t.Fatalf("RegisterDevice() error = %v", err)
	}
	if rec.Device.ID != "test-device-id" {
		t.Errorf("Device.ID = %q, want %q", rec.Device.ID, "test-device-id")
	}
	if rec.PrivateKey == "" || rec.PublicKey == "" {
		t.Error("PrivateKey or PublicKey is empty")
	}

	// Build outbound
	outbound, err := svc.BuildWarpOutbound("", 0, 0)
	if err != nil {
		t.Fatalf("BuildWarpOutbound() error = %v", err)
	}
	if outbound["type"] != "wireguard" {
		t.Errorf("outbound type = %q, want %q", outbound["type"], "wireguard")
	}
	if outbound["tag"] != "proxy_out" {
		t.Errorf("outbound tag = %q, want %q", outbound["tag"], "proxy_out")
	}

	// Verify file was saved
	if _, err := os.Stat(filepath.Join(tmpDir, "warp-account.json")); os.IsNotExist(err) {
		t.Error("warp-account.json not created")
	}
}

// TestRegisterDevice_noServer tests that RegisterDevice returns an error when no server is reachable.
func TestRegisterDevice_noServer(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)
	_, err := svc.RegisterDevice()
	if err == nil {
		t.Error("RegisterDevice() expected error with no server")
	}
}
