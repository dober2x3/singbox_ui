package certificate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandler_GenerateSelfSignedCert(t *testing.T) {
	svc := NewService(t.TempDir())
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/certificate", strings.NewReader(`{"domain":"test.com","valid_days":30}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateSelfSignedCert(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp CertificateInfo
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.CommonName != "test.com" {
		t.Errorf("CommonName = %q, want %q", resp.CommonName, "test.com")
	}
}

func TestHandler_GenerateSelfSignedCert_badRequest(t *testing.T) {
	svc := NewService(t.TempDir())
	h := NewHandler(svc)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/certificate", strings.NewReader(`invalid json`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.GenerateSelfSignedCert(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
