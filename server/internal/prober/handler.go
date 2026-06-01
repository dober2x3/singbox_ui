package prober

import (
	"net/http"

	"singbox-config-service/internal/docs"
	"singbox-config-service/internal/pkg/types"

	"github.com/gin-gonic/gin"
)

// ProberNodeRequest represents a single node to be added via the API.
type ProberNodeRequest struct {
	Tag      string `json:"tag" binding:"required"`
	Protocol string `json:"protocol" binding:"required"`
	Address  string `json:"address" binding:"required"`
	Port     int    `json:"port" binding:"required"`
}

// ProberNodesRequest represents a batch of nodes to update via the API.
type ProberNodesRequest struct {
	Nodes []ProberNodeRequest `json:"nodes" binding:"required"`
}

// Handler serves HTTP endpoints for prober operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Handler with the given prober service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetProberStatus returns the prober status and statistics.
// @Summary      Get prober status
// @Description  Returns the current prober engine status including running state and probe counts
// @Tags         prober
// @Produce      json
// @Success      200  {object}  ProberStatus
// @Router       /prober/status [get]
func (h *Handler) GetProberStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetStats())
}

// GetProbeResults returns all probe results.
// @Summary      Get all probe results
// @Description  Returns probe results for all registered nodes
// @Tags         prober
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /prober/results [get]
func (h *Handler) GetProbeResults(c *gin.Context) {
	results := h.service.GetAllResults()
	c.JSON(http.StatusOK, gin.H{
		"count":   len(results),
		"results": results,
	})
}

// GetProbeResult returns the probe result for a specific node.
// @Summary      Get probe result by tag
// @Description  Returns the probe result for a specific node identified by its tag
// @Tags         prober
// @Produce      json
// @Param        tag path string true "Node tag"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      404  {object}  docs.ErrorResponse
// @Router       /prober/results/{tag} [get]
func (h *Handler) GetProbeResult(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Node tag is required",
			Message: "The tag path parameter must not be empty",
		})
		return
	}

	result := h.service.GetResult(tag)
	if result == nil {
		c.JSON(http.StatusNotFound, docs.ErrorResponse{
			Error:   "Node not found",
			Message: "No probe result found for the given tag",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBestNode returns the online node with the lowest latency.
// @Summary      Get best node
// @Description  Returns the online node with the lowest measured latency
// @Tags         prober
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  docs.ErrorResponse
// @Router       /prober/best [get]
func (h *Handler) GetBestNode(c *gin.Context) {
	best := h.service.GetBestNode()
	if best == nil {
		c.JSON(http.StatusNotFound, docs.ErrorResponse{
			Error:   "No online nodes available",
			Message: "All nodes are offline or no nodes registered",
		})
		return
	}

	c.JSON(http.StatusOK, best)
}

// GetOnlineNodes returns all online nodes.
// @Summary      Get online nodes
// @Description  Returns all nodes that are currently online
// @Tags         prober
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /prober/online [get]
func (h *Handler) GetOnlineNodes(c *gin.Context) {
	online := h.service.GetOnlineNodes()
	c.JSON(http.StatusOK, gin.H{
		"count": len(online),
		"nodes": online,
	})
}

// AddProberNode adds a new node for probing.
// @Summary      Add prober node
// @Description  Adds a single new node to the prober's node list
// @Tags         prober
// @Accept       json
// @Produce      json
// @Param        request body ProberNodeRequest true "Node details"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /prober/nodes [post]
func (h *Handler) AddProberNode(c *gin.Context) {
	var req ProberNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	h.service.AddNode(types.ProbeNode{
		Tag:      req.Tag,
		Protocol: req.Protocol,
		Address:  req.Address,
		Port:     req.Port,
	})

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Node added successfully",
	})
}

// UpdateProberNodes replaces all probed nodes with the given list.
// @Summary      Update prober nodes
// @Description  Replaces the entire set of probed nodes with the given list
// @Tags         prober
// @Accept       json
// @Produce      json
// @Param        request body ProberNodesRequest true "List of nodes"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /prober/nodes [put]
func (h *Handler) UpdateProberNodes(c *gin.Context) {
	var req ProberNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	nodes := make([]types.ProbeNode, len(req.Nodes))
	for i, n := range req.Nodes {
		nodes[i] = types.ProbeNode{
			Tag:      n.Tag,
			Protocol: n.Protocol,
			Address:  n.Address,
			Port:     n.Port,
		}
	}

	h.service.UpdateNodes(nodes)

	if err := h.service.SaveNodes(); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to save nodes",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Nodes updated successfully",
	})
}

// RemoveProberNode removes a node from probing by tag.
// @Summary      Remove prober node
// @Description  Removes a single node from the prober's node list by tag
// @Tags         prober
// @Produce      json
// @Param        tag path string true "Node tag"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /prober/nodes/{tag} [delete]
func (h *Handler) RemoveProberNode(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Node tag is required",
			Message: "The tag path parameter must not be empty",
		})
		return
	}

	h.service.RemoveNode(tag)

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Node removed successfully",
	})
}

// ClearProberNodes removes all probed nodes.
// @Summary      Clear prober nodes
// @Description  Removes all nodes from the prober's node list
// @Tags         prober
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Router       /prober/nodes [delete]
func (h *Handler) ClearProberNodes(c *gin.Context) {
	h.service.ClearNodes()
	c.JSON(http.StatusOK, MessageResponse{Message: "All nodes cleared"})
}

// StartProber starts the probe loop.
// @Summary      Start prober
// @Description  Starts the periodic probing loop
// @Tags         prober
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Router       /prober/start [post]
func (h *Handler) StartProber(c *gin.Context) {
	if h.service.IsRunning() {
		c.JSON(http.StatusOK, MessageResponse{Message: "Prober is already running"})
		return
	}

	h.service.Start()
	c.JSON(http.StatusOK, MessageResponse{Message: "Prober started"})
}

// StopProber stops the probe loop.
// @Summary      Stop prober
// @Description  Stops the periodic probing loop
// @Tags         prober
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Router       /prober/stop [post]
func (h *Handler) StopProber(c *gin.Context) {
	if !h.service.IsRunning() {
		c.JSON(http.StatusOK, MessageResponse{Message: "Prober is not running"})
		return
	}

	h.service.Stop()
	c.JSON(http.StatusOK, MessageResponse{Message: "Prober stopped"})
}

// SyncNodesFromSubscription imports nodes from the subscription service and starts probing.
// @Summary      Sync nodes from subscription
// @Description  Imports proxy nodes from the subscription service and adds them to the prober
// @Tags         prober
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /prober/sync [post]
func (h *Handler) SyncNodesFromSubscription(c *gin.Context) {
	_, err := h.service.SyncNodesFromSubscription()
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Failed to sync nodes",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Nodes synced from subscription",
	})
}

// SaveProbeResultsToSubscription persists probe results to subscriptions.
// @Summary      Save probe results
// @Description  Saves probe results to the subscription service
// @Tags         prober
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /prober/save [post]
func (h *Handler) SaveProbeResultsToSubscription(c *gin.Context) {
	_, err := h.service.SaveProbeResults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to save probe results",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Probe results saved to subscription",
	})
}
