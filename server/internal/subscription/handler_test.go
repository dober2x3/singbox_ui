package subscription

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestHandler creates a Handler backed by a temporary directory for testing.
func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	dir := t.TempDir()
	store := NewFileStore(dir)
	svc := NewService(store)
	return NewHandler(svc)
}

// TestHandler_GetSubscriptions_Empty verifies that GET returns an empty list with count 0 initially.
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

// TestHandler_GetUserAgents verifies that the user-agents endpoint returns 5 predefined options.
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

// TestHandler_DeleteSubscription_NoID verifies that DELETE without an ID returns a 404.
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

// TestHandler_RefreshSubscription_NoID verifies that POST refresh without an ID returns 400.
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

// TestHandler_UpdateSubscriptionSettings_InvalidJSON verifies that PATCH with bad JSON returns 400.
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

// TestHandler_GetAllNodes_Empty verifies that GET nodes returns empty list with no subscriptions.
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

// TestHandler_RefreshAllSubscriptions_Empty verifies that POST refresh-all returns 200 even with no subscriptions.
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

// TestHandler_AddSubscription_MissingFields verifies that POST with empty JSON body returns 400.
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
