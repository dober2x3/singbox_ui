package speedtest

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/start", h.StartSpeedTest)
	rg.GET("/status", h.GetSpeedTestStatus)
	rg.POST("/stop", h.StopSpeedTest)
}
