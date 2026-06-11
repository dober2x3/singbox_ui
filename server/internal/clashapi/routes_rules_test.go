package clashapi

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRulesAndModeRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm := NewPortManager(9090)
	pm.Assign("default")

	r := gin.New()
	group := r.Group("/api/clash/instances/:name")
	RegisterRulesRoutes(group, pm)
	RegisterModeRoutes(group, pm)

	tests := []string{
		"/api/clash/instances/default/rules",
		"/api/clash/instances/default/mode",
	}
	for _, path := range tests {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == 404 {
			t.Fatalf("route not found: %s", path)
		}
	}
}
