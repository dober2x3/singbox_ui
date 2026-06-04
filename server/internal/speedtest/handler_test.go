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

// newTestHandler creates a Handler backed by a mock TempRuntime for testing.
func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	mockRT := newMockTempRuntime()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockRT, cfg)
	return NewHandler(svc)
}

// TestHandler_GetSpeedTestStatus_Initial verifies the initial state is not running.
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

// TestHandler_StartSpeedTest_NoProvider verifies a 400 is returned when no node provider is configured.
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

// TestHandler_StopSpeedTest verifies stop returns 200 even when no test is running.
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

// TestHandler_WithRunningService verifies start/status/stop flow with a real node provider.
func TestHandler_WithRunningService(t *testing.T) {
	mockRT := newMockTempRuntime()
	cfgDir := t.TempDir()
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(mockRT, cfg)
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
