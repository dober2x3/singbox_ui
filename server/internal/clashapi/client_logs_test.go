package clashapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLogLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"level": "info"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	level, err := c.GetLogLevel()
	if err != nil {
		t.Fatal(err)
	}
	if level != "info" {
		t.Fatalf("expected info, got %s", level)
	}
}

func TestSetLogLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["level"] != "debug" {
			t.Fatalf("expected level=debug, got %s", body["level"])
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.SetLogLevel("debug"); err != nil {
		t.Fatal(err)
	}
}
