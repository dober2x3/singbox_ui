package clashapi

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTrafficAndLogsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm := NewPortManager(9090)
	pm.Assign("default")

	r := gin.New()
	group := r.Group("/api/clash/instances/:name")
	RegisterTrafficRoutes(group, pm)
	RegisterLogsRoutes(group, pm)

	tests := []string{
		"/api/clash/instances/default/traffic",
		"/api/clash/instances/default/memory",
		"/api/clash/instances/default/logs",
		"/api/clash/instances/default/logs/level",
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
