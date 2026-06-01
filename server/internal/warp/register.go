package warp

import "github.com/gin-gonic/gin"

// RegisterRoutes registers WARP-related routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/account", h.GetWarpAccount)
	rg.DELETE("/account", h.DeleteWarpAccount)
	rg.POST("/register", h.RegisterWarp)
	rg.POST("/license", h.BindWarpLicense)
	rg.POST("/scan", h.ScanWarpEndpoints)
}
