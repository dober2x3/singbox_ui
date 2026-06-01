package prober

import (
	"net/http"

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

// GetProberStatus returns the current prober status and stats.
func (h *Handler) GetProberStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetStats())
}

// GetProbeResults returns all probe results.
func (h *Handler) GetProbeResults(c *gin.Context) {
	results := h.service.GetAllResults()
	c.JSON(http.StatusOK, gin.H{
		"count":   len(results),
		"results": results,
	})
}

// GetProbeResult returns the probe result for a specific node tag.
func (h *Handler) GetProbeResult(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Node tag is required"})
		return
	}

	result := h.service.GetResult(tag)
	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBestNode returns the online node with the lowest latency.
func (h *Handler) GetBestNode(c *gin.Context) {
	best := h.service.GetBestNode()
	if best == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No online nodes available",
			"message": "All nodes are offline or no nodes registered",
		})
		return
	}

	c.JSON(http.StatusOK, best)
}

// GetOnlineNodes returns all online nodes.
func (h *Handler) GetOnlineNodes(c *gin.Context) {
	online := h.service.GetOnlineNodes()
	c.JSON(http.StatusOK, gin.H{
		"count": len(online),
		"nodes": online,
	})
}

// AddProberNode adds a new node for probing.
func (h *Handler) AddProberNode(c *gin.Context) {
	var req ProberNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.service.AddNode(types.ProbeNode{
		Tag:      req.Tag,
		Protocol: req.Protocol,
		Address:  req.Address,
		Port:     req.Port,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Node added successfully",
		"tag":     req.Tag,
	})
}

// UpdateProberNodes replaces all probed nodes with the given list.
func (h *Handler) UpdateProberNodes(c *gin.Context) {
	var req ProberNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save nodes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nodes updated successfully",
		"count":   len(nodes),
	})
}

// RemoveProberNode removes a node from probing by tag.
func (h *Handler) RemoveProberNode(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Node tag is required"})
		return
	}

	h.service.RemoveNode(tag)

	c.JSON(http.StatusOK, gin.H{
		"message": "Node removed successfully",
		"tag":     tag,
	})
}

// ClearProberNodes removes all probed nodes.
func (h *Handler) ClearProberNodes(c *gin.Context) {
	h.service.ClearNodes()
	c.JSON(http.StatusOK, gin.H{"message": "All nodes cleared"})
}

// StartProber starts the probe loop.
func (h *Handler) StartProber(c *gin.Context) {
	if h.service.IsRunning() {
		c.JSON(http.StatusOK, gin.H{"message": "Prober is already running"})
		return
	}

	h.service.Start()
	c.JSON(http.StatusOK, gin.H{"message": "Prober started"})
}

// StopProber stops the probe loop.
func (h *Handler) StopProber(c *gin.Context) {
	if !h.service.IsRunning() {
		c.JSON(http.StatusOK, gin.H{"message": "Prober is not running"})
		return
	}

	h.service.Stop()
	c.JSON(http.StatusOK, gin.H{"message": "Prober stopped"})
}

// SyncNodesFromSubscription imports nodes from the subscription service and starts probing.
func (h *Handler) SyncNodesFromSubscription(c *gin.Context) {
	nodes, err := h.service.SyncNodesFromSubscription()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nodes synced from subscription",
		"nodeCount": len(nodes),
	})
}

// SaveProbeResultsToSubscription persists probe results to the subscription service.
func (h *Handler) SaveProbeResultsToSubscription(c *gin.Context) {
	count, err := h.service.SaveProbeResults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save probe results: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Probe results saved to subscription",
		"count":   count,
	})
}
