package handlers

import (
	"net"
	"net/http"
	"singbox-config-service/services"

	"github.com/gin-gonic/gin"
)

// GenerateWireGuardKeys generate WireGuard key pair
// @Summary Generate WireGuard key pair
// @Description Generate a WireGuard private and public key pair, IP address must be specified
// @Tags WireGuard
// @Accept json
// @Produce json
// @Param request body object{ip:string} true "Request params, must include IP address"
// @Success 200 {object} services.KeyCacheResponse
// @Failure 400 {object} ErrorResponse "IP address not specified"
// @Failure 500 {object} ErrorResponse "IP already exists or generation failed"
// @Router /api/wireguard/keygen [post]
func GenerateWireGuardKeys(c *gin.Context) {
	var request struct {
		IP string `json:"ip"` // full IP must be specified, e.g. "10.10.0.5"
	}

	if err := c.ShouldBindJSON(&request); err != nil || request.IP == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "IP address is required",
		})
		return
	}

	if net.ParseIP(request.IP) == nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid IP format",
			Message: "Please provide a valid IPv4 or IPv6 address",
		})
		return
	}

	// generate key pair using cache
	result, err := services.GenerateWireGuardKeysWithCache(request.IP)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate WireGuard keys",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ErrorResponse error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SaveClientConfig save client config
func SaveClientConfig(c *gin.Context) {
	configData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to read config data",
			Message: err.Error(),
		})
		return
	}

	err = services.SaveClientConfig(configData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save client config",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Client config saved successfully",
	})
}

// GetClientConfig get client config
func GetClientConfig(c *gin.Context) {
	data, err := services.GetClientConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Client config not found",
			Message: err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// GetPublicKeyFromPrivate derive public key from private key
func GetPublicKeyFromPrivate(c *gin.Context) {
	var request struct {
		PrivateKey string `json:"privateKey"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	publicKey, err := services.GeneratePublicKeyFromPrivate(request.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to generate public key",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"publicKey": publicKey,
	})
}

// SaveClientConfigFile save client config file
func SaveClientConfigFile(c *gin.Context) {
	var request struct {
		ClientIndex   int    `json:"clientIndex"`
		ConfigContent string `json:"configContent"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	err := services.SaveClientConfigFile(request.ClientIndex, request.ConfigContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save client config file",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Client config file saved successfully",
	})
}

// ListClientConfigFiles list all client config files
func ListClientConfigFiles(c *gin.Context) {
	configs, err := services.ListClientConfigFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list client config files",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// HealthCheck health check
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Sing-box Config Service is running",
	})
}

// GetKeysCache get keys cache list
func GetKeysCache(c *gin.Context) {
	cache, err := services.GetKeysCache()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get keys cache",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, cache)
}

// GetPublicIP get server public IP address
func GetPublicIP(c *gin.Context) {
	ip, err := services.GetPublicIP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get public IP",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ip": ip,
	})
}
