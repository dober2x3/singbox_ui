package certificate

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for certificate operations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GenerateSelfSignedCert generates a self-signed certificate for the given domain and validity period.
func (h *Handler) GenerateSelfSignedCert(c *gin.Context) {
	var req struct {
		Domain    string `json:"domain"`
		ValidDays int    `json:"valid_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	info, err := h.svc.GenerateSelfSignedCert(req.Domain, req.ValidDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// GetCertificateInfo returns information about the currently stored certificate.
func (h *Handler) GetCertificateInfo(c *gin.Context) {
	certPath := filepath.Join(h.svc.certDir, "cert.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}
	info, err := h.svc.GetCertificateInfo(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// UploadCertificate handles uploading cert.pem and key.pem files.
func (h *Handler) UploadCertificate(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	certFile, _, err := c.Request.FormFile("cert")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cert file required"})
		return
	}
	defer certFile.Close()

	keyFile, _, err := c.Request.FormFile("key")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key file required"})
		return
	}
	defer keyFile.Close()

	certData, err := io.ReadAll(certFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read cert file"})
		return
	}
	keyData, err := io.ReadAll(keyFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read key file"})
		return
	}

	certPath := filepath.Join(h.svc.certDir, "cert.pem")
	if err := os.WriteFile(certPath, certData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save cert"})
		return
	}
	keyPath := filepath.Join(h.svc.certDir, "key.pem")
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save key"})
		return
	}

	info, err := h.svc.GetCertificateInfo(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}
