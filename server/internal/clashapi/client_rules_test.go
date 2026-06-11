package clashapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(RulesResponse{
			Rules: []Rule{
				{Type: "Domain", Payload: "example.com", Proxy: "proxy"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, err := c.GetRules()
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
	}
}

func TestGetMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ConfigResponse{Mode: "Rule"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	mode, err := c.GetMode()
	if err != nil {
		t.Fatal(err)
	}
	if mode != "Rule" {
		t.Fatalf("expected Rule, got %s", mode)
	}
}

func TestSetMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["mode"] != "global" {
			t.Fatalf("expected mode=global, got %s", body["mode"])
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.SetMode("global"); err != nil {
		t.Fatal(err)
	}
}
