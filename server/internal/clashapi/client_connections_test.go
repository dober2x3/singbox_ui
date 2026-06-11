package clashapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetConnections(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ConnectionsResponse{
			DownloadTotal: 1000,
			UploadTotal:   500,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	resp, err := c.GetConnections()
	if err != nil {
		t.Fatal(err)
	}
	if resp.DownloadTotal != 1000 {
		t.Fatalf("expected 1000, got %d", resp.DownloadTotal)
	}
}

func TestCloseAllConnections(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/connections" {
			called = true
			w.WriteHeader(204)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.CloseAllConnections(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected DELETE /connections to be called")
	}
}

func TestCloseConnection(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/connections/abc123" {
			called = true
			w.WriteHeader(204)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "")
	if err := c.CloseConnection("abc123"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected DELETE /connections/abc123 to be called")
	}
}
