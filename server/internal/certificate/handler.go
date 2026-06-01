package certificate

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"singbox-config-service/internal/docs"
)

// Handler handles HTTP requests for certificate operations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GenerateSelfSignedCert generates a self-signed TLS certificate.
// @Summary      Generate self-signed certificate
// @Description  Generates a self-signed TLS certificate for the given domain and validity period, saves it to the singbox directory
// @Tags         certificate
// @Accept       json
// @Produce      json
// @Param        request body GenerateCertRequest true "Domain and validity period"
// @Success      200  {object}  CertificateInfo
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/certificate [post]
func (h *Handler) GenerateSelfSignedCert(c *gin.Context) {
	var req struct {
		Domain    string `json:"domain"`
		ValidDays int    `json:"valid_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: err.Error()})
		return
	}
	info, err := h.svc.GenerateSelfSignedCert(req.Domain, req.ValidDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// GetCertificateInfo returns information about the currently stored certificate.
// @Summary      Get certificate info
// @Description  Returns metadata about the stored TLS certificate including validity and fingerprint
// @Tags         certificate
// @Produce      json
// @Success      200  {object}  CertificateInfo
// @Failure      404  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/certificate [get]
func (h *Handler) GetCertificateInfo(c *gin.Context) {
	certPath := filepath.Join(h.svc.certDir, "cert.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, docs.ErrorResponse{Error: "certificate not found"})
		return
	}
	info, err := h.svc.GetCertificateInfo(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// UploadCertificate handles uploading cert.pem and key.pem files.
// @Summary      Upload certificate files
// @Description  Uploads cert.pem and key.pem files as multipart form data
// @Tags         certificate
// @Accept       mpfd
// @Produce      json
// @Param        cert formData file true "Certificate file (cert.pem)"
// @Param        key formData file true "Private key file (key.pem)"
// @Success      200  {object}  CertificateInfo
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/certificate/upload [post]
func (h *Handler) UploadCertificate(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "failed to parse form"})
		return
	}

	certFile, _, err := c.Request.FormFile("cert")
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "cert file required"})
		return
	}
	defer certFile.Close()

	keyFile, _, err := c.Request.FormFile("key")
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "key file required"})
		return
	}
	defer keyFile.Close()

	certData, err := io.ReadAll(certFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "failed to read cert file"})
		return
	}
	keyData, err := io.ReadAll(keyFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "failed to read key file"})
		return
	}

	certPath := filepath.Join(h.svc.certDir, "cert.pem")
	if err := os.WriteFile(certPath, certData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: "failed to save cert"})
		return
	}
	keyPath := filepath.Join(h.svc.certDir, "key.pem")
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: "failed to save key"})
		return
	}

	info, err := h.svc.GetCertificateInfo(certPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}
