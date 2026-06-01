package wireguard

import "github.com/gin-gonic/gin"

// RegisterRoutes registers WireGuard-related routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/keygen", h.GenerateWireGuardKeys)
	rg.POST("/pubkey", h.GetPublicKeyFromPrivate)
	rg.GET("/keys-cache", h.GetKeysCache)
	rg.GET("/public-ip", h.GetPublicIP)
	rg.GET("/client-config", h.GetClientConfig)
	rg.POST("/client-config", h.SaveClientConfig)
	rg.POST("/save-client-file", h.SaveClientConfigFile)
	rg.GET("/client-files", h.ListClientConfigFiles)
}
