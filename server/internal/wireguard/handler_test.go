package wireguard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHandler_GenerateWireGuardKeys tests successful key generation via the handler.
func TestHandler_GenerateWireGuardKeys(t *testing.T) {
	svc := NewService(t.TempDir())
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/keygen", strings.NewReader(`{"ip":"10.0.0.1"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateWireGuardKeys(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp KeyCacheResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want %q", resp.IP, "10.0.0.1")
	}
	if resp.PrivateKey == "" || resp.PublicKey == "" {
		t.Error("PrivateKey or PublicKey is empty")
	}
}

// TestHandler_GetPublicKeyFromPrivate tests deriving a public key from a private key via the handler.
func TestHandler_GetPublicKeyFromPrivate(t *testing.T) {
	svc := NewService(t.TempDir())
	h := NewHandler(svc)
	priv, _ := svc.GeneratePrivateKey()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/pubkey", strings.NewReader(`{"private_key":"`+priv+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GetPublicKeyFromPrivate(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.PublicKey == "" {
		t.Error("PublicKey is empty")
	}
}

// TestHandler_GetPublicKeyFromPrivate_missing tests that the handler returns 400 when the private key is missing.
func TestHandler_GetPublicKeyFromPrivate_missing(t *testing.T) {
	svc := NewService(t.TempDir())
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/pubkey", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GetPublicKeyFromPrivate(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
