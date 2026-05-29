package handlers

import (
	"context"
	"io"
	"net/http"
	"singbox-config-service/services"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// warpScanInFlight global scan mutex flag: prevent concurrent scans from causing exponential TCP dials
var warpScanInFlight atomic.Bool

// warpRegisterInFlight global register mutex flag: prevent double-click/concurrent orphan accounts,
// because CF /reg creates a new device on each call, duplicate registration wastes quota and invalidates
// the previous bearer token (the previous warp-account.json gets overwritten by the later write).
var warpRegisterInFlight atomic.Bool

// warpRegisterMaxBody max register request body size: prevent malicious large bodies from consuming memory
const warpRegisterMaxBody = 8 * 1024

// warpAccountView account info exposed to the frontend (without token)
type warpAccountView struct {
	Exists    bool   `json:"exists"`
	ID        string `json:"id,omitempty"`
	License   string `json:"license,omitempty"`
	Type      string `json:"type,omitempty"`
	WarpPlus  bool   `json:"warp_plus,omitempty"`
	V4        string `json:"v4,omitempty"`
	V6        string `json:"v6,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func recordToView(rec *services.WarpRecord) warpAccountView {
	if rec == nil {
		return warpAccountView{Exists: false}
	}
	return warpAccountView{
		Exists:    true,
		ID:        rec.Device.ID,
		License:   rec.Device.Account.License,
		Type:      rec.Device.Account.AccountType,
		WarpPlus:  rec.Device.Account.WarpPlus,
		V4:        rec.Device.Config.Interface.Addresses.V4,
		V6:        rec.Device.Config.Interface.Addresses.V6,
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
	}
}

// GetWarpAccount GET /api/warp/account — query locally cached WARP account
func GetWarpAccount(c *gin.Context) {
	rec, err := services.LoadWarpRecord()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recordToView(rec))
}

// DeleteWarpAccount DELETE /api/warp/account — delete local WARP account cache
func DeleteWarpAccount(c *gin.Context) {
	if err := services.DeleteWarpRecord(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// WarpRegisterRequest POST /api/warp/register request body
type WarpRegisterRequest struct {
	Force        bool   `json:"force"`         // true: ignore cache, force re-register
	License      string `json:"license"`       // optional: if provided, also bind WARP+ license
	EndpointHost string `json:"endpoint_host"` // optional: custom peer host (default engage.cloudflareclient.com)
	EndpointPort int    `json:"endpoint_port"` // optional: custom peer port (default 2408)
	MTU          int    `json:"mtu"`           // optional: MTU (default 1280)
}

// WarpRegisterResponse POST /api/warp/register response body
type WarpRegisterResponse struct {
	Account  warpAccountView        `json:"account"`
	Outbound map[string]interface{} `json:"outbound"`
}

// RegisterWarp POST /api/warp/register — register WARP and generate outbound config
func RegisterWarp(c *gin.Context) {
	// concurrency mutex: prevent frontend double-click or multi-tab concurrent calls from creating multiple orphan devices on CF,
	// the later written warp-account.json would overwrite the previous bearer token, invalidating it.
	if !warpRegisterInFlight.CompareAndSwap(false, true) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "register already running"})
		return
	}
	defer warpRegisterInFlight.Store(false)

	// body size limit: WarpRegisterRequest only has a few fields, 8KB is well above the reasonable limit
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, warpRegisterMaxBody)

	var req WarpRegisterRequest
	// accept empty body (io.EOF), but reject malformed JSON — otherwise fields would be silently zeroed
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body: " + err.Error()})
		return
	}

	// read cache first; also validate cache integrity, corrupted cache is treated as unregistered
	var rec *services.WarpRecord
	if !req.Force {
		r, err := services.LoadWarpRecord()
		if err == nil && r != nil && r.Device.ID != "" && len(r.Device.Config.Peers) > 0 {
			rec = r
		}
	}

	// cache missing or forced, register new device
	if rec == nil {
		r, err := services.RegisterWarpDevice()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rec = r
	}

	// if license is provided, bind WARP+
	if req.License != "" && rec.Device.Account.License != req.License {
		r, err := services.BindWarpLicense(rec, req.License)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		rec = r
	}

	outbound, err := services.BuildWarpOutbound(rec, req.EndpointHost, req.EndpointPort, req.MTU)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, WarpRegisterResponse{
		Account:  recordToView(rec),
		Outbound: outbound,
	})
}

// WarpBindLicenseRequest POST /api/warp/license request body
type WarpBindLicenseRequest struct {
	License string `json:"license"`
}

// BindWarpLicense POST /api/warp/license — bind WARP+ license to a registered device
func BindWarpLicense(c *gin.Context) {
	// body size limit: license string is very short, 8KB is enough to defend against malicious large packets
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, warpRegisterMaxBody)

	var req WarpBindLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.License == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license cannot be empty"})
		return
	}
	rec, err := services.LoadWarpRecord()
	if err != nil || rec == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please register a WARP device first"})
		return
	}
	rec, err = services.BindWarpLicense(rec, req.License)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recordToView(rec))
}

// ScanWarpEndpoints POST /api/warp/scan — scan available Cloudflare edge endpoints
func ScanWarpEndpoints(c *gin.Context) {
	// scan mutex: prevent concurrent requests from amplifying TCP dial count
	if !warpScanInFlight.CompareAndSwap(false, true) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "scan already running"})
		return
	}
	defer warpScanInFlight.Store(false)

	// body size limit: scan request only contains a few integers, 8KB is enough
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, warpRegisterMaxBody)

	cfg := services.DefaultWarpScanConfig()

	var body struct {
		SamplePerRange int `json:"sample_per_range"`
		TimeoutMs      int `json:"timeout_ms"`
		TopN           int `json:"top_n"`
	}
	if err := c.ShouldBindJSON(&body); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body: " + err.Error()})
		return
	}
	// return explicit 400 on out-of-bounds parameters instead of silently falling back to defaults — otherwise users won't notice when they increase parameters past valid ranges
	if body.SamplePerRange != 0 {
		if body.SamplePerRange < 1 || body.SamplePerRange > 32 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sample_per_range must be in [1, 32]"})
			return
		}
		cfg.SamplePerRange = body.SamplePerRange
	}
	if body.TimeoutMs != 0 {
		if body.TimeoutMs < 100 || body.TimeoutMs > 5000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "timeout_ms must be in [100, 5000]"})
			return
		}
		cfg.Timeout = time.Duration(body.TimeoutMs) * time.Millisecond
	}
	if body.TopN != 0 {
		if body.TopN < 1 || body.TopN > 32 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "top_n must be in [1, 32]"})
			return
		}
		cfg.TopN = body.TopN
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	results, err := services.ScanWarpEndpoints(ctx, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"endpoints": results,
		"ports":     services.WarpEndpointPorts(),
	})
}
