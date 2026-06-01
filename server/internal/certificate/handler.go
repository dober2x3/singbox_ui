package certificate

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

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

func (h *Handler) UploadCertificate(c *gin.Context) {
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

	certData, _ := io.ReadAll(certFile)
	keyData, _ := io.ReadAll(keyFile)

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
		c.JSON(http.StatusOK, gin.H{"message": "Files saved but could not read info"})
		return
	}
	c.JSON(http.StatusOK, info)
}
