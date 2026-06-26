package clashapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProxiesRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pm := NewPortManager(9090)
	pm.Assign("default")

	r := gin.New()
	group := r.Group("/api/clash/instances/:name")
	RegisterProxiesRoutes(group, pm)

	clashSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ProxiesResponse{
			Proxies: map[string]ProxyGroup{
				"Proxy": {Type: "Selector", Now: "NodeSG", All: []string{"NodeSG", "NodeJP"}},
			},
		})
	}))
	defer clashSrv.Close()

	req := httptest.NewRequest("GET", "/api/clash/instances/default/proxies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == 404 {
		t.Fatal("route not found (404)")
	}
}
