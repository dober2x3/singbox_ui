package clashapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientGetProxies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies" {
			t.Fatalf("expected /proxies, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Fatalf("expected Bearer test-secret, got %s", r.Header.Get("Authorization"))
		}
		if err := json.NewEncoder(w).Encode(ProxiesResponse{
			Proxies: map[string]ProxyGroup{
				"Proxy": {Type: "Selector", Now: "NodeSG", All: []string{"NodeSG", "NodeJP"}},
			},
		}); err != nil {
			t.Fatal(err)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-secret")
	resp, err := c.GetProxies()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Proxies["Proxy"].Now != "NodeSG" {
		t.Fatalf("expected NodeSG, got %s", resp.Proxies["Proxy"].Now)
	}
}
