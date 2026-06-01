package certificate

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/certificate", h.GenerateSelfSignedCert)
	rg.GET("/certificate", h.GetCertificateInfo)
	rg.POST("/certificate/upload", h.UploadCertificate)
}
