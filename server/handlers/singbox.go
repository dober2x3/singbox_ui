package handlers

import (
	"log"
	"net/http"
	"regexp"
	"singbox-config-service/services"

	"github.com/gin-gonic/gin"
)

// validNamePattern valid instance name: letters, digits, underscore, hyphen
var validNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z_-]{1,9}$`)

// validateName validate instance name: 2-10 English letters
func validateName(c *gin.Context) (string, bool) {
	name := c.Param("name")
	if name == "" || !validNamePattern.MatchString(name) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid config name",
			Message: "Name must be 2-10 chars, start with letter, allow letters, underscore, hyphen",
		})
		return "", false
	}
	return name, true
}

// GetSingboxVersion get sing-box version
func GetSingboxVersion(c *gin.Context) {
	version, err := services.GetSingBoxVersion()
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "sing-box not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"version": version,
	})
}

// RunSingbox run sing-box container
func RunSingbox(c *gin.Context) {
	// request parameters no longer needed, config file uses default path

	containerID, err := services.RunSingboxContainer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to start sing-box container",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "sing-box container started successfully",
		"containerId": containerID,
	})
}

// SaveConfig save config file
func SaveConfig(c *gin.Context) {
	configData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to read config data",
			Message: err.Error(),
		})
		return
	}

	configPath, err := services.SaveConfig(configData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save config",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Config saved successfully",
		"path":    configPath,
	})
}

// GetConfig get config file
func GetConfig(c *gin.Context) {
	data, err := services.GetConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Config not found",
			Message: err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// StopSingbox stop sing-box container
func StopSingbox(c *gin.Context) {
	// PID parameter no longer needed

	if err := services.StopSingboxContainer(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to stop sing-box container",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "sing-box container stopped successfully",
	})
}

// GetSingboxLogs get sing-box logs
func GetSingboxLogs(c *gin.Context) {
	logs := services.GetContainerLogs()
	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}

// CheckSingboxStatus check if sing-box container is running
func CheckSingboxStatus(c *gin.Context) {
	// PID parameter no longer needed

	running, containerID := services.CheckContainerRunning()

	c.JSON(http.StatusOK, gin.H{
		"running":     running,
		"containerId": containerID,
	})
}

// EnsureImage ensure sing-box image exists
func EnsureImage(c *gin.Context) {
	if err := services.EnsureSingboxImage(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to ensure sing-box image",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "sing-box image is ready",
	})
}

// ========== Multi-config multi-container API ==========

// ListNamedConfigs list all named configs and their container status
func ListNamedConfigs(c *gin.Context) {
	configs, err := services.ListNamedConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list configs",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
	})
}

// CheckNamedConfig validate named config
func CheckNamedConfig(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	valid, output, err := services.CheckNamedConfig(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to check config",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   valid,
		"message": output,
	})
}

// SaveNamedConfigWithContainer save config to named directory and validate (for multi-container scenario)
func SaveNamedConfigWithContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	configData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to read config data",
			Message: err.Error(),
		})
		return
	}

	// save config file first
	if err := services.SaveNamedConfigWithDir(name, configData); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save config",
			Message: err.Error(),
		})
		return
	}

	// validate config after saving
	valid, output, err := services.CheckNamedConfig(name)
	if err != nil {
		// validation failure does not affect saving, but returns a warning
		c.JSON(http.StatusOK, gin.H{
			"message": "Config saved but validation unavailable",
			"name":    name,
			"valid":   nil,
			"warning": err.Error(),
		})
		return
	}

	if !valid {
		c.JSON(http.StatusOK, gin.H{
			"message": output,
			"name":    name,
			"valid":   false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Config saved and validated successfully",
		"name":    name,
		"valid":   true,
	})
}

// LoadNamedConfigFromContainer load config from named directory
func LoadNamedConfigFromContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	data, err := services.LoadNamedConfigFromDir(name)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Config not found",
			Message: err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// DeleteNamedConfigWithContainer delete named config and its container
func DeleteNamedConfigWithContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	if err := services.DeleteNamedConfigWithDir(name); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete config",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Config deleted successfully",
		"name":    name,
	})
}

// RunNamedContainer start container for named config
func RunNamedContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	containerID, err := services.RunNamedContainer(name)
	if err != nil {
		log.Printf("Failed to start container for %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to start container",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Container started successfully",
		"name":        name,
		"containerId": containerID,
	})
}

// StopNamedContainer stop container for named config
func StopNamedContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	if err := services.StopNamedContainer(name); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to stop container",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Container stopped successfully",
		"name":    name,
	})
}

// GetNamedContainerStatus get named container status
func GetNamedContainerStatus(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	running, containerID := services.GetNamedContainerStatus(name)

	c.JSON(http.StatusOK, gin.H{
		"name":        name,
		"running":     running,
		"containerId": containerID,
	})
}

// GetNamedContainerLogs get named container logs
func GetNamedContainerLogs(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	logs := services.GetNamedContainerLogs(name)
	c.JSON(http.StatusOK, gin.H{
		"name": name,
		"logs": logs,
	})
}

// ListAllContainers list all sing-box containers
func ListAllContainers(c *gin.Context) {
	containers, err := services.ListAllContainers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list containers",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"containers": containers,
	})
}
