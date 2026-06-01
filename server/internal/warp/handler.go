package warp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"singbox-config-service/internal/docs"
)

// Handler handles HTTP requests for WARP-related endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GetWarpAccount returns the stored WARP device account info.
// @Summary      Get WARP account
// @Description  Returns the stored WARP device account info
// @Tags         warp
// @Produce      json
// @Success      200  {object}  WarpRecord
// @Failure      404  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /warp/account [get]
func (h *Handler) GetWarpAccount(c *gin.Context) {
	rec, err := h.svc.LoadRecord()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, docs.ErrorResponse{Error: "no WARP device registered"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// DeleteWarpAccount deletes the stored WARP device account.
// @Summary      Delete WARP account
// @Description  Deletes the stored WARP device account
// @Tags         warp
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /warp/account [delete]
func (h *Handler) DeleteWarpAccount(c *gin.Context) {
	if err := h.svc.DeleteRecord(); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "WARP account deleted"})
}

// RegisterWarp registers a new WARP device with Cloudflare.
// @Summary      Register WARP device
// @Description  Registers a new WARP device with Cloudflare
// @Tags         warp
// @Produce      json
// @Success      200  {object}  WarpRegisterResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /warp/register [post]
func (h *Handler) RegisterWarp(c *gin.Context) {
	rec, err := h.svc.RegisterDevice()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// BindWarpLicense binds a WARP+ license key to the registered device.
// @Summary      Bind WARP+ license
// @Description  Binds a WARP+ license key to the registered device
// @Tags         warp
// @Accept       json
// @Produce      json
// @Param        request body LicenseBindRequest true "License key"
// @Success      200  {object}  WarpRecord
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /warp/license [post]
func (h *Handler) BindWarpLicense(c *gin.Context) {
	var req LicenseBindRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.License == "" {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{Error: "license is required"})
		return
	}
	rec, err := h.svc.BindLicense(req.License)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, rec)
}

// ScanWarpEndpoints scans for available WARP endpoints with best latency.
// @Summary      Scan WARP endpoints
// @Description  Scans for available WARP endpoints with best latency
// @Tags         warp
// @Accept       json
// @Produce      json
// @Param        config body WarpScanConfig false "Scan configuration (uses defaults if omitted)"
// @Success      200  {array}   WarpEndpointResult
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /warp/scan [post]
func (h *Handler) ScanWarpEndpoints(c *gin.Context) {
	cfg := DefaultWarpScanConfig()
	_ = c.ShouldBindJSON(&cfg) // Use whatever the client sent
	results, err := ScanWarpEndpoints(c.Request.Context(), cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, results)
}
