package speedtest

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

func (h *Handler) StartSpeedTest(c *gin.Context) {
	if err := h.svc.StartSpeedTest(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "speed test started"})
}

func (h *Handler) GetSpeedTestStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.GetSpeedTestState())
}

func (h *Handler) StopSpeedTest(c *gin.Context) {
	h.svc.StopSpeedTest()
	c.JSON(http.StatusOK, gin.H{"message": "stop requested"})
}
