package warp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandler_RegisterWarp(t *testing.T) {
	// Mock Cloudflare API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/v0a2158/reg" {
			_ = json.NewEncoder(w).Encode(WarpRegisterResponse{
				ID:    "test-device-id",
				Token: "test-token",
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

	svc := NewService(t.TempDir())
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/register", nil)

	h.RegisterWarp(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandler_GetWarpAccount_notFound(t *testing.T) {
	svc := NewService(t.TempDir())
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/account", nil)

	h.GetWarpAccount(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
