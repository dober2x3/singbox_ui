package resourcecheck

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/types"
)

func tempDirWithRetryCleanup(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "resourcecheck-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 3; i++ {
			if err := os.RemoveAll(dir); err == nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
	return dir
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	cfgDir := tempDirWithRetryCleanup(t)
	t.Setenv("DATA_DIR", cfgDir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatal(err)
	}

	checker := NewChecker(&mockRunner{}, cfg)
	np := &mockNodeProvider{
		nodes: []types.ProxyNode{
			{
				Name: "test", Protocol: "vmess", Address: "1.1.1.1", Port: 443,
				Outbound: map[string]interface{}{"type": "vmess", "server": "1.1.1.1", "server_port": 443},
			},
		},
	}

	svc := NewService(checker, np, ProberConfig{
		DBPath: filepath.Join(cfgDir, "test.db"),
	})
	return NewHandler(svc)
}

func TestHandler_GetResources_Empty(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/resourcecheck/resources", h.GetResources)

	req := httptest.NewRequest("GET", "/api/resourcecheck/resources", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp["count"].(float64) != 0 {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestHandler_GetStatus_Idle(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/resourcecheck/status", h.GetStatus)

	req := httptest.NewRequest("GET", "/api/resourcecheck/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var status CheckStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if status.Running {
		t.Error("expected not running")
	}
}

func TestHandler_Run_NoNodes(t *testing.T) {
	cfgDir := tempDirWithRetryCleanup(t)
	t.Setenv("DATA_DIR", cfgDir)
	cfg, _ := config.Init()
	checker := NewChecker(&mockRunner{}, cfg)
	svc := NewService(checker, &mockNodeProvider{}, ProberConfig{
		DBPath: filepath.Join(cfgDir, "test.db"),
	})
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/resourcecheck/run", h.Run)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/resourcecheck/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Stop_Idempotent(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/resourcecheck/stop", h.Stop)

	req := httptest.NewRequest("POST", "/api/resourcecheck/stop", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_Schedule(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/resourcecheck/schedule", h.Schedule)

	body := `{"interval_sec": 3600}`
	req := httptest.NewRequest("POST", "/api/resourcecheck/schedule", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body = `{"interval_sec": 0}`
	req = httptest.NewRequest("POST", "/api/resourcecheck/schedule", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_GetResults_Empty(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/resourcecheck/results", h.GetResults)

	req := httptest.NewRequest("GET", "/api/resourcecheck/results", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	count := resp["count"].(float64)
	if count != 0 {
		t.Errorf("expected count 0, got %v", count)
	}
}

func TestHandler_Reload_NoFile(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/resourcecheck/reload", h.Reload)

	req := httptest.NewRequest("POST", "/api/resourcecheck/reload", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
