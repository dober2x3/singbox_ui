package speedtest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	_ "singbox-config-service/internal/docs"
)

// Handler handles HTTP requests for speed test operations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// StartSpeedTest starts a speed test.
// @Summary      Start speed test
// @Description  Starts a speed test for all loaded proxy nodes
// @Tags         speedtest
// @Produce      json
// @Success      200  {object}  map[string]string  "speed test started"
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /speedtest/start [post]
func (h *Handler) StartSpeedTest(c *gin.Context) {
	if err := h.svc.StartSpeedTest(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "speed test started"})
}

// GetSpeedTestStatus returns the current speed test state.
// @Summary      Get speed test status
// @Description  Returns the current speed test state including progress and results
// @Tags         speedtest
// @Produce      json
// @Success      200  {object}  SpeedTestState
// @Router       /speedtest/status [get]
func (h *Handler) GetSpeedTestStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.GetSpeedTestState())
}

// StopSpeedTest stops a running speed test.
// @Summary      Stop speed test
// @Description  Stops the currently running speed test if one is in progress
// @Tags         speedtest
// @Produce      json
// @Success      200  {object}  map[string]string  "stop requested"
// @Router       /speedtest/stop [post]
func (h *Handler) StopSpeedTest(c *gin.Context) {
	h.svc.StopSpeedTest()
	c.JSON(http.StatusOK, gin.H{"message": "stop requested"})
}
