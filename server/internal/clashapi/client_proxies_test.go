package clashapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProxies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/proxies" {
			t.Fatalf("expected /proxies, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ProxiesResponse{
			Proxies: map[string]ProxyGroup{
				"Proxy": {Type: "Selector", Now: "NodeSG", All: []string{"NodeSG", "NodeJP"}},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, err := c.GetProxies()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Proxies["Proxy"].Now != "NodeSG" {
		t.Fatalf("expected NodeSG, got %s", resp.Proxies["Proxy"].Now)
	}
}

func TestSwitchProxy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/proxies/Proxy" {
			t.Fatalf("expected /proxies/Proxy, got %s", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["name"] != "NodeSG" {
			t.Fatalf("expected name=NodeSG, got %s", body["name"])
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.SwitchProxy("Proxy", "NodeSG"); err != nil {
		t.Fatal(err)
	}
}

func TestGetProxyDelay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies/NodeSG/delay" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		urlParam := r.URL.Query().Get("url")
		if urlParam == "" {
			t.Fatal("expected url param")
		}
		_ = json.NewEncoder(w).Encode(map[string]int{"NodeSG": 123})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	delay, err := c.GetProxyDelay("NodeSG", "https://www.gstatic.com/generate_204", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if delay != 123 {
		t.Fatalf("expected 123, got %d", delay)
	}
}

func TestGetProxyDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies/NodeSG" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ProxyDetail{Type: "Shadowsocks", Name: "NodeSG"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	detail, err := c.GetProxy("NodeSG")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Type != "Shadowsocks" {
		t.Fatalf("expected Shadowsocks, got %s", detail.Type)
	}
}

func TestGetProxyDetailWithSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Fatalf("expected Bearer test-secret, got %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(ProxyDetail{Type: "Shadowsocks", Name: "NodeSG"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-secret")
	detail, err := c.GetProxy("NodeSG")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Type != "Shadowsocks" {
		t.Fatalf("expected Shadowsocks, got %s", detail.Type)
	}
}
