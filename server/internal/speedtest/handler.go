package speedtest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for speed test operations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// StartSpeedTest starts a speed test. Returns 400 if the test cannot be started.
func (h *Handler) StartSpeedTest(c *gin.Context) {
	if err := h.svc.StartSpeedTest(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "speed test started"})
}

// GetSpeedTestStatus returns the current speed test state as JSON.
func (h *Handler) GetSpeedTestStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.GetSpeedTestState())
}

// StopSpeedTest cancels a running speed test.
func (h *Handler) StopSpeedTest(c *gin.Context) {
	h.svc.StopSpeedTest()
	c.JSON(http.StatusOK, gin.H{"message": "stop requested"})
}
