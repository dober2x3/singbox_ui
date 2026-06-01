package subscription

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"singbox-config-service/internal/pkg/types"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func (h *Handler) GetSubscriptions(c *gin.Context) {
	subData, err := h.svc.GetAllSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error:   "Failed to load subscriptions",
			Message: err.Error(),
		})
		return
	}

	totalNodes := 0
	for _, sub := range subData.Subscriptions {
		totalNodes += len(sub.Nodes)
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriptions": subData.Subscriptions,
		"count":         len(subData.Subscriptions),
		"totalNodes":    totalNodes,
	})
}

func (h *Handler) AddSubscription(c *gin.Context) {
	var request struct {
		Name      string `json:"name" binding:"required"`
		URL       string `json:"url" binding:"required"`
		UserAgent string `json:"user_agent"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	entry, err := h.svc.AddSubscription(request.Name, request.URL, request.UserAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error:   "Failed to add subscription",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Subscription added successfully",
		"subscription": entry,
		"nodeCount":    len(entry.Nodes),
	})
}

func (h *Handler) RefreshSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "Invalid request",
			Message: "subscription id is required",
		})
		return
	}

	entry, err := h.svc.UpdateSubscription(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error:   "Failed to refresh subscription",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Subscription refreshed successfully",
		"subscription": entry,
		"nodeCount":    len(entry.Nodes),
	})
}

func (h *Handler) DeleteSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, errorResponse{
			Error:   "Invalid request",
			Message: "subscription id is required",
		})
		return
	}

	if err := h.svc.DeleteSubscription(id); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error:   "Failed to delete subscription",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscription deleted successfully",
	})
}

func (h *Handler) RefreshAllSubscriptions(c *gin.Context) {
	data, err := h.svc.RefreshAllSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error:   "Failed to refresh subscriptions",
			Message: err.Error(),
		})
		return
	}

	totalNodes := 0
	for _, sub := range data.Subscriptions {
		totalNodes += len(sub.Nodes)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "All subscriptions refreshed",
		"subscriptions": data.Subscriptions,
		"count":         len(data.Subscriptions),
		"totalNodes":    totalNodes,
	})
}

func (h *Handler) UpdateSubscriptionSettings(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "Invalid request", Message: "subscription id is required"})
		return
	}

	var request struct {
		AutoUpdate     bool `json:"auto_update"`
		UpdateInterval int  `json:"update_interval"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	entry, err := h.svc.UpdateSubscriptionSettings(id, request.AutoUpdate, request.UpdateInterval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "Failed to update settings", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Settings updated",
		"subscription": entry,
	})
}

func (h *Handler) GetAllNodes(c *gin.Context) {
	nodes, err := h.svc.GetAllNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error:   "Failed to get nodes",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"count": len(nodes),
	})
}

func (h *Handler) GetUserAgents(c *gin.Context) {
	type UAOption struct {
		Key   string `json:"key"`
		Label string `json:"label"`
		Value string `json:"value"`
	}

	options := []UAOption{
		{Key: "default", Label: "Default Browser", Value: types.PredefinedUserAgents["default"]},
		{Key: "clash-verge", Label: "Clash Verge", Value: types.PredefinedUserAgents["clash-verge"]},
		{Key: "clash-meta", Label: "Clash Meta", Value: types.PredefinedUserAgents["clash-meta"]},
		{Key: "v2rayn", Label: "v2rayN", Value: types.PredefinedUserAgents["v2rayn"]},
		{Key: "v2rayng", Label: "v2rayNG", Value: types.PredefinedUserAgents["v2rayng"]},
	}

	c.JSON(http.StatusOK, gin.H{
		"user_agents": options,
	})
}
