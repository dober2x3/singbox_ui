package resourcecheck

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetResources(c *gin.Context) {
	resources := h.svc.GetResources()
	c.JSON(http.StatusOK, gin.H{
		"count":     len(resources),
		"resources": resources,
	})
}

func (h *Handler) GetResults(c *gin.Context) {
	results, err := h.svc.GetLatestResults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	c.JSON(http.StatusOK, gin.H{
		"count":   len(results),
		"results": results,
	})
}

func (h *Handler) GetResultsForTag(c *gin.Context) {
	tag := c.Param("tag")
	results, err := h.svc.GetResultsForTag(tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	c.JSON(http.StatusOK, gin.H{
		"tag":     tag,
		"count":   len(results),
		"results": results,
	})
}

func (h *Handler) GetHistory(c *gin.Context) {
	resource := c.Param("resource")
	tag := c.Param("tag")
	results, err := h.svc.GetHistory(resource, tag, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if results == nil {
		results = []CheckResult{}
	}
	c.JSON(http.StatusOK, gin.H{
		"resource": resource,
		"tag":      tag,
		"count":    len(results),
		"results":  results,
	})
}

func (h *Handler) Run(c *gin.Context) {
	status := h.svc.GetStatus()
	if status.Running {
		c.JSON(http.StatusConflict, gin.H{"error": "check already running"})
		return
	}

	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = RunRequest{}
	}

	var runErr error
	if req.Tag != "" {
		runErr = h.svc.RunForTag(c.Request.Context(), req.Tag)
	} else {
		runErr = h.svc.RunAll(c.Request.Context())
	}

	if runErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": runErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "check completed"})
}

func (h *Handler) Stop(c *gin.Context) {
	h.svc.Stop()
	c.JSON(http.StatusOK, gin.H{"message": "stop requested"})
}

func (h *Handler) Schedule(c *gin.Context) {
	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IntervalSec <= 0 {
		h.svc.StopScheduler()
		c.JSON(http.StatusOK, gin.H{"message": "scheduler stopped"})
		return
	}

	h.svc.StartScheduler(req.IntervalSec)
	c.JSON(http.StatusOK, gin.H{
		"message":      "scheduler started",
		"interval_sec": req.IntervalSec,
	})
}

func (h *Handler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.GetStatus())
}

func (h *Handler) Reload(c *gin.Context) {
	if err := h.svc.ReloadResources(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resources reloaded"})
}
