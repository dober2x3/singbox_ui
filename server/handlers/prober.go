package handlers

import (
	"net/http"
	"singbox-config-service/services"

	"github.com/gin-gonic/gin"
)

// ProberNodeRequest add node request
type ProberNodeRequest struct {
	Tag      string `json:"tag" binding:"required"`
	Protocol string `json:"protocol" binding:"required"`
	Address  string `json:"address" binding:"required"`
	Port     int    `json:"port" binding:"required"`
}

// ProberNodesRequest batch add nodes request
type ProberNodesRequest struct {
	Nodes []ProberNodeRequest `json:"nodes" binding:"required"`
}

// GetProberStatus get prober status
func GetProberStatus(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	c.JSON(http.StatusOK, prober.GetStats())
}

// GetProbeResults get all probe results
func GetProbeResults(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	results := prober.GetAllResults()
	c.JSON(http.StatusOK, gin.H{
		"count":   len(results),
		"results": results,
	})
}

// GetProbeResult get single node probe result
func GetProbeResult(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Node tag is required",
		})
		return
	}

	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	result := prober.GetResult(tag)
	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Node not found",
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetBestNode get best node (lowest latency)
func GetBestNode(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	best := prober.GetBestNode()
	if best == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No online nodes available",
			"message": "All nodes are offline or no nodes registered",
		})
		return
	}

	c.JSON(http.StatusOK, best)
}

// GetOnlineNodes get all online nodes
func GetOnlineNodes(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	online := prober.GetOnlineNodes()
	c.JSON(http.StatusOK, gin.H{
		"count": len(online),
		"nodes": online,
	})
}

// AddProberNode add probe node
func AddProberNode(c *gin.Context) {
	var req ProberNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	prober.AddNode(services.ProbeNode{
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

// UpdateProberNodes batch update probe nodes
func UpdateProberNodes(c *gin.Context) {
	var req ProberNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	nodes := make([]services.ProbeNode, len(req.Nodes))
	for i, n := range req.Nodes {
		nodes[i] = services.ProbeNode{
			Tag:      n.Tag,
			Protocol: n.Protocol,
			Address:  n.Address,
			Port:     n.Port,
		}
	}

	prober.UpdateNodes(nodes)

	// save to file
	if err := prober.SaveNodesToFile(); err != nil {
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

// RemoveProberNode remove probe node
func RemoveProberNode(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Node tag is required",
		})
		return
	}

	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	prober.RemoveNode(tag)

	c.JSON(http.StatusOK, gin.H{
		"message": "Node removed successfully",
		"tag":     tag,
	})
}

// ClearProberNodes clear all probe nodes
func ClearProberNodes(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	prober.ClearNodes()

	c.JSON(http.StatusOK, gin.H{
		"message": "All nodes cleared",
	})
}

// StartProber start prober
func StartProber(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	if prober.IsRunning() {
		c.JSON(http.StatusOK, gin.H{
			"message": "Prober is already running",
		})
		return
	}

	prober.Start()

	c.JSON(http.StatusOK, gin.H{
		"message": "Prober started",
	})
}

// StopProber stop prober
func StopProber(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	if !prober.IsRunning() {
		c.JSON(http.StatusOK, gin.H{
			"message": "Prober is not running",
		})
		return
	}

	prober.Stop()

	c.JSON(http.StatusOK, gin.H{
		"message": "Prober stopped",
	})
}

// SyncNodesFromSubscription sync nodes from subscription data to prober
func SyncNodesFromSubscription(c *gin.Context) {
	// load all nodes from all subscriptions
	allNodes, err := services.GetAllNodes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No subscription data found: " + err.Error(),
		})
		return
	}

	if len(allNodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No nodes in subscription",
		})
		return
	}

	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	// convert node format, use the tag already generated in outbound
	nodes := make([]services.ProbeNode, 0, len(allNodes))
	for _, n := range allNodes {
		// use the tag from outbound (generated by sanitizeTag in subscription.go)
		// this ensures prober and subscription use the same tag
		tag := ""
		if outbound := n.Outbound; outbound != nil {
			if t, ok := outbound["tag"].(string); ok {
				tag = t
			}
		}
		// if outbound has no tag, generate using sanitizeTag logic
		if tag == "" {
			tag = services.SanitizeTag(n.Protocol, n.Address, n.Port)
		}

		nodes = append(nodes, services.ProbeNode{
			Tag:      tag,
			Protocol: n.Protocol,
			Address:  n.Address,
			Port:     n.Port,
		})
	}

	prober.UpdateNodes(nodes)

	// if prober is not running, auto-start it
	if !prober.IsRunning() {
		prober.Start()
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nodes synced from subscription",
		"nodeCount": len(nodes),
	})
}

// SaveProbeResultsToSubscription save probe results to subscription file
func SaveProbeResultsToSubscription(c *gin.Context) {
	prober := services.GetProber()
	if prober == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Prober not initialized",
		})
		return
	}

	// get all probe results
	results := prober.GetAllResults()
	if len(results) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No probe results to save",
			"count":   0,
		})
		return
	}

	// convert to update format
	updates := make([]services.ProbeResultUpdate, 0, len(results))
	for _, r := range results {
		updates = append(updates, services.ProbeResultUpdate{
			Tag:         r.NodeTag,
			Latency:     r.Latency,
			Online:      r.Status == "online",
			LastProbe:   r.LastProbe.Format("2006-01-02 15:04:05"),
			SuccessRate: int(r.SuccessRate),
		})
	}

	// save to subscription file
	if err := services.UpdateProbeResults(updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save probe results: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Probe results saved to subscription",
		"count":   len(updates),
	})
}
