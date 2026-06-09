package resourcecheck

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/resources", h.GetResources)
	rg.GET("/results", h.GetResults)
	rg.GET("/results/:tag", h.GetResultsForTag)
	rg.GET("/history/:resource/:tag", h.GetHistory)
	rg.POST("/run", h.Run)
	rg.POST("/stop", h.Stop)
	rg.POST("/schedule", h.Schedule)
	rg.GET("/status", h.GetStatus)
	rg.POST("/reload", h.Reload)
}
