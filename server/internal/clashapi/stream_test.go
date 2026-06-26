package clashapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamHandlerSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer srv.Close()

	_ = NewClient(srv.URL, "")
}
