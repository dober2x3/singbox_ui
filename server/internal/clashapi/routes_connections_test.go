package clashapi

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestConnectionsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm := NewPortManager(9090)
	pm.Assign("default")

	r := gin.New()
	group := r.Group("/api/clash/instances/:name")
	RegisterConnectionsRoutes(group, pm)

	req := httptest.NewRequest("GET", "/api/clash/instances/default/connections", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == 404 {
		t.Fatal("route not found (404)")
	}
}
