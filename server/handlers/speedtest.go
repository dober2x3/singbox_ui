package handlers

import (
	"net/http"
	"singbox-config-service/services"

	"github.com/gin-gonic/gin"
)

// StartSpeedTest start proxy speed test (serial test all subscription nodes)
func StartSpeedTest(c *gin.Context) {
	if err := services.StartSpeedTest(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "speed test started"})
}

// GetSpeedTestStatus get current speed test status and results
func GetSpeedTestStatus(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetSpeedTestState())
}

// StopSpeedTest cancel running speed test
func StopSpeedTest(c *gin.Context) {
	services.StopSpeedTest()
	c.JSON(http.StatusOK, gin.H{"message": "stop requested"})
}
