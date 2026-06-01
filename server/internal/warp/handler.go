package warp

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

func (h *Handler) GetWarpAccount(c *gin.Context) {
	rec, err := h.svc.LoadRecord()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no WARP device registered"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *Handler) DeleteWarpAccount(c *gin.Context) {
	if err := h.svc.DeleteRecord(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WARP account deleted"})
}

func (h *Handler) RegisterWarp(c *gin.Context) {
	rec, err := h.svc.RegisterDevice()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *Handler) BindWarpLicense(c *gin.Context) {
	var req struct {
		License string `json:"license"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.License == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license is required"})
		return
	}
	rec, err := h.svc.BindLicense(req.License)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *Handler) ScanWarpEndpoints(c *gin.Context) {
	cfg := DefaultWarpScanConfig()
	_ = c.ShouldBindJSON(&cfg) // Use whatever the client sent
	results, err := ScanWarpEndpoints(c.Request.Context(), cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}
