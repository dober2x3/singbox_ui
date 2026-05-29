package handlers

import (
	"net/http"
	"singbox-config-service/services"

	"github.com/gin-gonic/gin"
)

// GetSubscriptions get all subscriptions
func GetSubscriptions(c *gin.Context) {
	subData, err := services.LoadSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load subscriptions",
			Message: err.Error(),
		})
		return
	}

	// calculate total node count
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

// AddSubscription add subscription
func AddSubscription(c *gin.Context) {
	var request struct {
		Name      string `json:"name" binding:"required"`
		URL       string `json:"url" binding:"required"`
		UserAgent string `json:"user_agent"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	entry, err := services.AddSubscription(request.Name, request.URL, request.UserAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
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

// RefreshSubscription refresh single subscription
func RefreshSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "subscription id is required",
		})
		return
	}

	entry, err := services.UpdateSubscription(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
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

// DeleteSubscription delete subscription
func DeleteSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "subscription id is required",
		})
		return
	}

	if err := services.DeleteSubscription(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete subscription",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscription deleted successfully",
	})
}

// RefreshAllSubscriptions refresh all subscriptions
func RefreshAllSubscriptions(c *gin.Context) {
	data, err := services.RefreshAllSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
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

// GetUserAgents get predefined User-Agent list
func GetUserAgents(c *gin.Context) {
	type UAOption struct {
		Key   string `json:"key"`
		Label string `json:"label"`
		Value string `json:"value"`
	}

	options := []UAOption{
		{Key: "default", Label: "Default Browser", Value: services.PredefinedUserAgents["default"]},
		{Key: "clash-verge", Label: "Clash Verge", Value: services.PredefinedUserAgents["clash-verge"]},
		{Key: "clash-meta", Label: "Clash Meta", Value: services.PredefinedUserAgents["clash-meta"]},
		{Key: "v2rayn", Label: "v2rayN", Value: services.PredefinedUserAgents["v2rayn"]},
		{Key: "v2rayng", Label: "v2rayNG", Value: services.PredefinedUserAgents["v2rayng"]},
	}

	c.JSON(http.StatusOK, gin.H{
		"user_agents": options,
	})
}

// UpdateSubscriptionSettings update subscription auto-update settings
func UpdateSubscriptionSettings(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: "subscription id is required"})
		return
	}

	var request struct {
		AutoUpdate     bool `json:"auto_update"`
		UpdateInterval int  `json:"update_interval"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	entry, err := services.UpdateSubscriptionSettings(id, request.AutoUpdate, request.UpdateInterval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update settings", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Settings updated",
		"subscription": entry,
	})
}

// GetAllNodes get all nodes
func GetAllNodes(c *gin.Context) {
	nodes, err := services.GetAllNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
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
