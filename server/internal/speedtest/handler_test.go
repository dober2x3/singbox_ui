package speedtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	mockDocker := newMockContainerAPI()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)
	return NewHandler(svc)
}

func TestHandler_GetSpeedTestStatus_Initial(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/speedtest/status", h.GetSpeedTestStatus)

	req := httptest.NewRequest("GET", "/api/speedtest/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var state SpeedTestState
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if state.Running {
		t.Error("expected state.Running = false initially")
	}
}

func TestHandler_StartSpeedTest_NoProvider(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/speedtest/start", h.StartSpeedTest)

	req := httptest.NewRequest("POST", "/api/speedtest/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_StopSpeedTest(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/speedtest/stop", h.StopSpeedTest)

	req := httptest.NewRequest("POST", "/api/speedtest/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_WithRunningService(t *testing.T) {
	mockDocker := newMockContainerAPI()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockDocker, cfg)
	svc.WithNodeProvider(&mockNodeProvider{
		nodes: []types.ProxyNode{
			{Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443}},
		},
	})
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/speedtest/start", h.StartSpeedTest)
	r.GET("/api/speedtest/status", h.GetSpeedTestStatus)
	r.POST("/api/speedtest/stop", h.StopSpeedTest)

	req := httptest.NewRequest("POST", "/api/speedtest/start", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("start expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/api/speedtest/status", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status expected 200, got %d", w.Code)
	}

	var state SpeedTestState
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !state.Running {
		t.Log("state not running yet (goroutine may not have started)")
	}
}
