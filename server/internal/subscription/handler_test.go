package subscription

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	dir := t.TempDir()
	store := NewFileStore(dir)
	svc := NewService(store)
	return NewHandler(svc)
}

func TestHandler_GetSubscriptions_Empty(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/subscription", h.GetSubscriptions)

	req := httptest.NewRequest("GET", "/api/subscription", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"] != float64(0) {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestHandler_GetUserAgents(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/subscription/user-agents", h.GetUserAgents)

	req := httptest.NewRequest("GET", "/api/subscription/user-agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	agents, ok := resp["user_agents"].([]interface{})
	if !ok {
		t.Fatal("expected user_agents array")
	}
	if len(agents) != 5 {
		t.Errorf("expected 5 user agents, got %d", len(agents))
	}
}

func TestHandler_DeleteSubscription_NoID(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/api/subscription/:id", h.DeleteSubscription)

	req := httptest.NewRequest("DELETE", "/api/subscription/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Route doesn't match / (no :id), returns 404
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandler_RefreshSubscription_NoID(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/subscription/:id/refresh", h.RefreshSubscription)

	req := httptest.NewRequest("POST", "/api/subscription//refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_UpdateSubscriptionSettings_InvalidJSON(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/api/subscription/:id/settings", h.UpdateSubscriptionSettings)

	body := strings.NewReader(`invalid json`)
	req := httptest.NewRequest("PATCH", "/api/subscription/test-id/settings", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_GetAllNodes_Empty(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/subscription/nodes", h.GetAllNodes)

	req := httptest.NewRequest("GET", "/api/subscription/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_RefreshAllSubscriptions_Empty(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/subscription/refresh-all", h.RefreshAllSubscriptions)

	req := httptest.NewRequest("POST", "/api/subscription/refresh-all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandler_AddSubscription_MissingFields(t *testing.T) {
	h := newTestHandler(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/subscription", h.AddSubscription)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest("POST", "/api/subscription", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
