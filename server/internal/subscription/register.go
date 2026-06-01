package subscription

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("", h.GetSubscriptions)
	rg.POST("", h.AddSubscription)
	rg.POST("/:id/refresh", h.RefreshSubscription)
	rg.PATCH("/:id/settings", h.UpdateSubscriptionSettings)
	rg.DELETE("/:id", h.DeleteSubscription)
	rg.POST("/refresh-all", h.RefreshAllSubscriptions)
	rg.GET("/nodes", h.GetAllNodes)
	rg.GET("/user-agents", h.GetUserAgents)
}
