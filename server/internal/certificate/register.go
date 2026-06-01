package certificate

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all certificate routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/certificate", h.GenerateSelfSignedCert)
	rg.GET("/certificate", h.GetCertificateInfo)
	rg.POST("/certificate/upload", h.UploadCertificate)
}
