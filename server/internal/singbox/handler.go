package singbox

import (
	"log"
	"net/http"
	"regexp"

	"singbox-config-service/internal/docs"

	"github.com/gin-gonic/gin"
)

// validNamePattern validates config names: 2-10 chars, start with letter, allow letters/underscore/hyphen.
var validNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z_-]{1,9}$`)

// validateName validates the "name" URL parameter against validNamePattern.
// Returns the name and true if valid, or sends an error response and returns "", false.
func validateName(c *gin.Context) (string, bool) {
	name := c.Param("name")
	if name == "" || !validNamePattern.MatchString(name) {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Invalid config name",
			Message: "Name must be 2-10 chars, start with letter, allow letters, underscore, hyphen",
		})
		return "", false
	}
	return name, true
}

// Handler handles HTTP requests for sing-box operations.
type Handler struct {
	service *Service
}

// NewHandler creates a new Handler with the given Service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetVersion returns the sing-box version.
// @Summary      Get sing-box version
// @Description  Returns the version of the installed sing-box Docker image
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  VersionResponse
// @Failure      404  {object}  docs.ErrorResponse
// @Router       /singbox/version [get]
func (h *Handler) GetVersion(c *gin.Context) {
	version, err := h.service.GetVersion()
	if err != nil {
		c.JSON(http.StatusNotFound, docs.ErrorResponse{
			Error:   "sing-box not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, VersionResponse{
		Version: version,
	})
}

// GetConfig returns the current sing-box configuration.
// @Summary      Get sing-box config
// @Description  Returns the current sing-box configuration as raw JSON
// @Tags         singbox
// @Produce      json
// @Success      200  {string}  string  "Raw config JSON"
// @Failure      404  {object}  docs.ErrorResponse
// @Router       /singbox/config [get]
func (h *Handler) GetConfig(c *gin.Context) {
	data, err := h.service.GetConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, docs.ErrorResponse{
			Error:   "Config not found",
			Message: err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// SaveConfig saves the sing-box configuration from the request body.
// @Summary      Save sing-box config
// @Description  Saves the sing-box configuration from the raw request body
// @Tags         singbox
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "config saved"
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/config [post]
func (h *Handler) SaveConfig(c *gin.Context) {
	configData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Failed to read config data",
			Message: err.Error(),
		})
		return
	}

	configPath, err := h.service.SaveConfig(configData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// RunSingbox starts the sing-box container.
// @Summary      Run sing-box
// @Description  Starts the sing-box Docker container with the current configuration
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "container started"
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/run [post]
func (h *Handler) RunSingbox(c *gin.Context) {
	containerID, err := h.service.RunContainer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// StopSingbox stops the sing-box container.
// @Summary      Stop sing-box
// @Description  Stops the running sing-box Docker container
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "container stopped"
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/stop [post]
func (h *Handler) StopSingbox(c *gin.Context) {
	if err := h.service.StopContainer(); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to stop sing-box container",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "sing-box container stopped successfully",
	})
}

// GetSingboxLogs returns the logs from the sing-box container.
// @Summary      Get sing-box logs
// @Description  Returns the logs from the sing-box Docker container
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  LogResponse
// @Router       /singbox/logs [get]
func (h *Handler) GetSingboxLogs(c *gin.Context) {
	logs := h.service.ContainerLogs()
	c.JSON(http.StatusOK, LogResponse{
		Logs: logs,
	})
}

// CheckSingboxStatus checks whether the sing-box container is running.
// @Summary      Check sing-box status
// @Description  Returns whether the sing-box container is currently running
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  StatusResponse
// @Router       /singbox/status [get]
func (h *Handler) CheckSingboxStatus(c *gin.Context) {
	running, containerID := h.service.ContainerStatus()

	c.JSON(http.StatusOK, StatusResponse{
		Running:     running,
		ContainerID: containerID,
	})
}

// EnsureImage ensures the sing-box Docker image is available.
// @Summary      Ensure sing-box image
// @Description  Pulls the sing-box Docker image if not already present
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "image ready"
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/ensure-image [post]
func (h *Handler) EnsureImage(c *gin.Context) {
	if err := h.service.EnsureImage(); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to ensure sing-box image",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "sing-box image is ready",
	})
}

// ListNamedConfigs returns all named configuration instances.
// @Summary      List named configs
// @Description  Returns all named configuration instances with their status
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "list of configs"
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/instances [get]
func (h *Handler) ListNamedConfigs(c *gin.Context) {
	configs, err := h.service.ListNamedConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to list configs",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs": configs,
	})
}

// SaveNamedConfigWithContainer saves a named configuration and validates it.
// @Summary      Save named config
// @Description  Saves a named configuration, validates it, and returns the result
// @Tags         singbox
// @Accept       json
// @Produce      json
// @Param        name path string true "Config name (2-10 chars, start with letter)"
// @Success      200  {object}  CheckConfigResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name}/config [post]
func (h *Handler) SaveNamedConfigWithContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	configData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, docs.ErrorResponse{
			Error:   "Failed to read config data",
			Message: err.Error(),
		})
		return
	}

	if err := h.service.SaveNamedConfig(name, configData); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to save config",
			Message: err.Error(),
		})
		return
	}

	valid, output := h.service.CheckNamedConfig(name)

	if !valid {
		c.JSON(http.StatusOK, CheckConfigResponse{
			Valid:   false,
			Message: output,
		})
		return
	}

	c.JSON(http.StatusOK, CheckConfigResponse{
		Valid:   true,
		Message: "Config saved and validated successfully",
	})
}

// LoadNamedConfigFromContainer returns a named configuration.
// @Summary      Get named config
// @Description  Returns a named configuration as raw JSON
// @Tags         singbox
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200  {string}  string  "Raw config JSON"
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      404  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name}/config [get]
func (h *Handler) LoadNamedConfigFromContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	data, err := h.service.LoadNamedConfig(name)
	if err != nil {
		c.JSON(http.StatusNotFound, docs.ErrorResponse{
			Error:   "Config not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config":     string(data),
		"clash_port": h.service.GetClashPort(name),
	})
}

// CheckNamedConfig validates a named configuration's JSON syntax.
// @Summary      Check named config
// @Description  Validates the JSON syntax of a named configuration
// @Tags         singbox
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200  {object}  CheckConfigResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name}/check [post]
func (h *Handler) CheckNamedConfig(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	valid, output := h.service.CheckNamedConfig(name)

	c.JSON(http.StatusOK, CheckConfigResponse{
		Valid:   valid,
		Message: output,
	})
}

// DeleteNamedConfigWithContainer deletes a named configuration instance.
// @Summary      Delete named config
// @Description  Deletes a named configuration and stops its container if running
// @Tags         singbox
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200  {object}  map[string]interface{}  "config deleted"
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name} [delete]
func (h *Handler) DeleteNamedConfigWithContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	if err := h.service.DeleteNamedConfig(name); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
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

// RunNamedContainer starts a named container instance.
// @Summary      Run named container
// @Description  Starts a named sing-box container instance
// @Tags         singbox
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200  {object}  NamedInstanceResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name}/run [post]
func (h *Handler) RunNamedContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	containerID, err := h.service.RunNamedContainer(name)
	if err != nil {
		log.Printf("Failed to start container for %s: %v", name, err)
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to start container",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, NamedInstanceResponse{
		Message:     "Container started successfully",
		Name:        name,
		ContainerID: containerID,
	})
}

// StopNamedContainer stops a named container instance.
// @Summary      Stop named container
// @Description  Stops a named sing-box container instance
// @Tags         singbox
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200  {object}  NamedInstanceResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name}/stop [post]
func (h *Handler) StopNamedContainer(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	if err := h.service.StopNamedContainer(name); err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to stop container",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, NamedInstanceResponse{
		Message: "Container stopped successfully",
		Name:    name,
	})
}

// GetNamedContainerStatus returns the status of a named container.
// @Summary      Get named container status
// @Description  Returns the running status and container ID of a named instance
// @Tags         singbox
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200  {object}  NamedInstanceResponse
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name}/status [get]
func (h *Handler) GetNamedContainerStatus(c *gin.Context) {
	name, ok := validateName(c)
	if !ok {
		return
	}

	_, containerID := h.service.NamedContainerStatus(name)

	c.JSON(http.StatusOK, NamedInstanceResponse{
		Name:        name,
		Message:     "OK",
		ContainerID: containerID,
	})
}

// GetNamedContainerLogs returns the logs for a named container.
// @Summary      Get named container logs
// @Description  Returns the logs for a named sing-box container instance
// @Tags         singbox
// @Produce      json
// @Param        name path string true "Config name"
// @Success      200  {object}  map[string]interface{}  "logs for container"
// @Failure      400  {object}  docs.ErrorResponse
// @Router       /singbox/instances/{name}/logs [get]
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

// ListAllContainers returns all sing-box containers.
// @Summary      List all containers
// @Description  Returns all sing-box Docker containers across all named instances
// @Tags         singbox
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "list of containers"
// @Failure      500  {object}  docs.ErrorResponse
// @Router       /singbox/containers [get]
func (h *Handler) ListAllContainers(c *gin.Context) {
	containers, err := h.service.ListAllContainers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, docs.ErrorResponse{
			Error:   "Failed to list containers",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"containers": containers,
	})
}
