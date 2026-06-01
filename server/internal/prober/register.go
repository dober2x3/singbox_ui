package prober

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all prober HTTP routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/status", h.GetProberStatus)
	rg.GET("/results", h.GetProbeResults)
	rg.GET("/results/:tag", h.GetProbeResult)
	rg.GET("/best", h.GetBestNode)
	rg.GET("/online", h.GetOnlineNodes)
	rg.POST("/nodes", h.AddProberNode)
	rg.PUT("/nodes", h.UpdateProberNodes)
	rg.DELETE("/nodes/:tag", h.RemoveProberNode)
	rg.DELETE("/nodes", h.ClearProberNodes)
	rg.POST("/start", h.StartProber)
	rg.POST("/stop", h.StopProber)
	rg.POST("/sync", h.SyncNodesFromSubscription)
	rg.POST("/save", h.SaveProbeResultsToSubscription)
}
