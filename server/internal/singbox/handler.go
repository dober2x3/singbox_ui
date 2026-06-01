package singbox

import (
	"log"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var validNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z_-]{1,9}$`)

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

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetVersion(c *gin.Context) {
	version, err := h.service.GetVersion()
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

func (h *Handler) GetConfig(c *gin.Context) {
	data, err := h.service.GetConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Config not found",
			Message: err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

func (h *Handler) SaveConfig(c *gin.Context) {
	configData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Failed to read config data",
			Message: err.Error(),
		})
		return
	}

	configPath, err := h.service.SaveConfig(configData)
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

func (h *Handler) RunSingbox(c *gin.Context) {
	containerID, err := h.service.RunContainer()
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

func (h *Handler) StopSingbox(c *gin.Context) {
	if err := h.service.StopContainer(); err != nil {
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

func (h *Handler) GetSingboxLogs(c *gin.Context) {
	logs := h.service.ContainerLogs()
	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}

func (h *Handler) CheckSingboxStatus(c *gin.Context) {
	running, containerID := h.service.ContainerStatus()

	c.JSON(http.StatusOK, gin.H{
		"running":     running,
		"containerId": containerID,
	})
}

func (h *Handler) EnsureImage(c *gin.Context) {
	if err := h.service.EnsureImage(); err != nil {
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

func (h *Handler) ListNamedConfigs(c *gin.Context) {
	configs, err := h.service.ListNamedConfigs()
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

func (h *Handler) SaveNamedConfigWithContainer(c *gin.Context) {
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

	if err := h.service.SaveNamedConfig(name, configData); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to save config",
			Message: err.Error(),
		})
		return
	}

	valid, output := h.service.CheckNamedConfig(name)

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

func (h *Handler) LoadNamedConfigFromContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	data, err := h.service.LoadNamedConfig(name)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Config not found",
			Message: err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

func (h *Handler) CheckNamedConfig(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	valid, output := h.service.CheckNamedConfig(name)

	c.JSON(http.StatusOK, gin.H{
		"valid":   valid,
		"message": output,
	})
}

func (h *Handler) DeleteNamedConfigWithContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	if err := h.service.DeleteNamedConfig(name); err != nil {
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

func (h *Handler) RunNamedContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	containerID, err := h.service.RunNamedContainer(name)
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

func (h *Handler) StopNamedContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	if err := h.service.StopNamedContainer(name); err != nil {
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

func (h *Handler) GetNamedContainerStatus(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	running, containerID := h.service.NamedContainerStatus(name)

	c.JSON(http.StatusOK, gin.H{
		"name":        name,
		"running":     running,
		"containerId": containerID,
	})
}

func (h *Handler) GetNamedContainerLogs(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	logs := h.service.NamedContainerLogs(name)
	c.JSON(http.StatusOK, gin.H{
		"name": name,
		"logs": logs,
	})
}

func (h *Handler) ListAllContainers(c *gin.Context) {
	containers, err := h.service.ListAllContainers()
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

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
