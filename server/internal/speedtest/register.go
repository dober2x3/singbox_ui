package speedtest

import "github.com/gin-gonic/gin"

// RegisterRoutes registers speed test endpoints under the given RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/start", h.StartSpeedTest)
	rg.GET("/status", h.GetSpeedTestStatus)
	rg.POST("/stop", h.StopSpeedTest)
}
