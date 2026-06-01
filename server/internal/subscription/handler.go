package subscription

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"singbox-config-service/internal/docs"
	"singbox-config-service/internal/pkg/types"
)

// Handler handles HTTP requests for subscription operations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetSubscriptions handles GET requests to list all subscriptions with total node counts.
// @Summary      Get all subscriptions
// @Description  Returns all subscriptions with total node counts
// @Tags         subscription
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /subscription [get]
func (h *Handler) GetSubscriptions(c *gin.Context) {
	subData, err := h.svc.GetAllSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// AddSubscription handles POST requests to add a new subscription and returns the parsed nodes.
// @Summary      Add a new subscription
// @Description  Adds a new subscription by URL and parses its proxy nodes
// @Tags         subscription
// @Accept       json
// @Produce      json
// @Param        request body AddSubscriptionRequest true "Subscription details"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /subscription [post]
func (h *Handler) AddSubscription(c *gin.Context) {
	var request AddSubscriptionRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	entry, err := h.svc.AddSubscription(request.Name, request.URL, request.UserAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// RefreshSubscription handles POST requests to re-fetch a single subscription by ID.
// @Summary      Refresh a single subscription
// @Description  Re-fetches and parses a subscription by its ID
// @Tags         subscription
// @Produce      json
// @Param        id path string true "Subscription ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /subscription/{id}/refresh [post]
func (h *Handler) RefreshSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Invalid request",
			Message: "subscription id is required",
		})
		return
	}

	entry, err := h.svc.UpdateSubscription(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// DeleteSubscription handles DELETE requests to remove a subscription by ID.
// @Summary      Delete a subscription
// @Description  Removes a subscription by its ID
// @Tags         subscription
// @Produce      json
// @Param        id path string true "Subscription ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /subscription/{id} [delete]
func (h *Handler) DeleteSubscription(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Invalid request",
			Message: "subscription id is required",
		})
		return
	}

	if err := h.svc.DeleteSubscription(id); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to delete subscription",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscription deleted successfully",
	})
}

// RefreshAllSubscriptions handles POST requests to re-fetch all subscriptions.
// @Summary      Refresh all subscriptions
// @Description  Re-fetches and parses all subscriptions
// @Tags         subscription
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /subscription/refresh-all [post]
func (h *Handler) RefreshAllSubscriptions(c *gin.Context) {
	data, err := h.svc.RefreshAllSubscriptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// UpdateSubscriptionSettings handles PATCH requests to update auto-update and interval settings.
// @Summary      Update subscription settings
// @Description  Updates auto-update and interval settings for a subscription
// @Tags         subscription
// @Accept       json
// @Produce      json
// @Param        id path string true "Subscription ID"
// @Param        request body UpdateSettingsRequest true "Settings"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /subscription/{id}/settings [patch]
func (h *Handler) UpdateSubscriptionSettings(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "Invalid request", Message: "subscription id is required"})
		return
	}

	var request UpdateSettingsRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	entry, err := h.svc.UpdateSubscriptionSettings(id, request.AutoUpdate, request.UpdateInterval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: "Failed to update settings", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Settings updated",
		"subscription": entry,
	})
}

// GetAllNodes handles GET requests to return all proxy nodes across all subscriptions.
// @Summary      Get all proxy nodes
// @Description  Returns all proxy nodes across all subscriptions
// @Tags         subscription
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /subscription/nodes [get]
func (h *Handler) GetAllNodes(c *gin.Context) {
	nodes, err := h.svc.GetAllNodes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// GetUserAgents handles GET requests to return available user-agent presets.
// @Summary      Get user agent presets
// @Description  Returns available user-agent presets for subscription fetching
// @Tags         subscription
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /subscription/user-agents [get]
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
