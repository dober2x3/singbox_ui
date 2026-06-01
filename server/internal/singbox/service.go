// Package singbox provides services for managing sing-box configuration and containers.
package singbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
)

// Service provides business logic for sing-box configuration and container management.
type Service struct {
	docker ContainerManager
	cfg    *config.Config
}

// NewService creates a new Service with the given ContainerManager and Config.
func NewService(docker ContainerManager, cfg *config.Config) *Service {
	return &Service{
		docker: docker,
		cfg:    cfg,
	}
}

// SaveConfig writes the configuration data to disk and returns the file path.
func (s *Service) SaveConfig(data []byte) (string, error) {
	configPath := filepath.Join(s.cfg.GetSingboxDir(), "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save config: %w", err)
	}
	return configPath, nil
}

// GetConfig reads and returns the configuration data from disk.
func (s *Service) GetConfig() ([]byte, error) {
	configPath := filepath.Join(s.cfg.GetSingboxDir(), "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	return data, nil
}

// RunContainer creates and starts a sing-box Docker container.
func (s *Service) RunContainer() (string, error) {
	configPath := filepath.Join(s.cfg.GetSingboxDir(), "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config file not found, please save config first")
	}

	hostConfigPath := configPath
	if s.cfg != nil {
		if resolved, err := s.cfg.ResolveHostConfigDir(configPath); err == nil {
			hostConfigPath = resolved
		}
	}

	containerConfig := map[string]interface{}{
		"Image": "sing-box",
		"Cmd":   []string{"run", "-c", "/etc/sing-box/config.json"},
		"ExposedPorts": map[string]interface{}{
			"1080/tcp": struct{}{},
			"1080/udp": struct{}{},
		},
	}
	hostConfig := map[string]interface{}{
		"Binds":        []string{hostConfigPath + ":/etc/sing-box/config.json:ro"},
		"NetworkMode":  "host",
		"CapAdd":       []string{"NET_ADMIN", "SYS_MODULE"},
		"PortBindings": map[string]interface{}{},
	}

	id, err := s.docker.ContainerCreate(context.TODO(), containerConfig, hostConfig, "singbox")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := s.docker.ContainerStart(context.TODO(), id); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return id, nil
}

// StopContainer stops the sing-box container.
func (s *Service) StopContainer() error {
	state, err := s.docker.GetContainerState(context.TODO(), "singbox")
	if err != nil {
		return err
	}
	if state == "" {
		return nil
	}
	timeout := 10
	return s.docker.ContainerStop(context.TODO(), "singbox", &timeout)
}

// ContainerStatus returns whether the sing-box container is running and its ID.
func (s *Service) ContainerStatus() (running bool, containerID string) {
	state, err := s.docker.GetContainerState(context.TODO(), "singbox")
	if err != nil || state == "" {
		return false, ""
	}
	return state == "running", state
}

// ContainerLogs returns the last 100 log lines from the sing-box container.
func (s *Service) ContainerLogs() string {
	logs, err := s.docker.ContainerLogs(context.TODO(), "singbox", "100")
	if err != nil {
		return fmt.Sprintf("Error getting logs: %v", err)
	}
	return logs
}

// EnsureImage ensures the sing-box Docker image is pulled and available.
func (s *Service) EnsureImage() error {
	return s.docker.EnsureImage(context.Background(), "ghcr.io/sagernet/sing-box:latest", "")
}

// GetVersion returns the sing-box version string.
func (s *Service) GetVersion() (string, error) {
	return "sing-box 1.10.0", nil
}

// getNamedConfigPath returns the filesystem path for a named config.
func (s *Service) getNamedConfigPath(name string) string {
	return filepath.Join(s.cfg.GetSingboxDir(), "instances", name, "config.json")
}

// SaveNamedConfig writes a named configuration to disk.
func (s *Service) SaveNamedConfig(name string, data []byte) error {
	path := s.getNamedConfigPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create instance directory: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadNamedConfig reads and returns a named configuration from disk.
func (s *Service) LoadNamedConfig(name string) ([]byte, error) {
	path := s.getNamedConfigPath(name)
	return os.ReadFile(path)
}

// DeleteNamedConfig removes a named configuration and its directory.
func (s *Service) DeleteNamedConfig(name string) error {
	path := filepath.Dir(s.getNamedConfigPath(name))
	return os.RemoveAll(path)
}

// ListNamedConfigs returns all named configuration instances with their running status.
func (s *Service) ListNamedConfigs() ([]NamedConfigInfo, error) {
	instancesDir := filepath.Join(s.cfg.GetSingboxDir(), "instances")
	if _, err := os.Stat(instancesDir); os.IsNotExist(err) {
		return []NamedConfigInfo{}, nil
	}
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return nil, err
	}
	var configs []NamedConfigInfo
	for _, entry := range entries {
		if entry.IsDir() {
			running, _ := s.NamedContainerStatus(entry.Name())
			configs = append(configs, NamedConfigInfo{
				Name:    entry.Name(),
				Running: running,
			})
		}
	}
	return configs, nil
}

// RunNamedContainer creates and starts a named sing-box container.
func (s *Service) RunNamedContainer(name string) (string, error) {
	configPath := s.getNamedConfigPath(name)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config for '%s' not found", name)
	}

	hostConfigPath := configPath
	if resolved, err := s.cfg.ResolveHostConfigDir(configPath); err == nil {
		hostConfigPath = resolved
	}

	containerName := "singbox-" + name
	state, _ := s.docker.GetContainerState(context.TODO(), containerName)
	if state == "running" {
		return state, fmt.Errorf("container %s is already running", containerName)
	}
	if state != "" {
		_ = s.docker.ContainerRemove(context.TODO(), containerName, true)
	}

	containerConfig := map[string]interface{}{
		"Image": "sing-box",
		"Cmd":   []string{"run", "-c", "/etc/sing-box/config.json"},
	}
	hostConfig := map[string]interface{}{
		"Binds":       []string{hostConfigPath + ":/etc/sing-box/config.json:ro"},
		"NetworkMode": "host",
		"CapAdd":      []string{"NET_ADMIN", "SYS_MODULE"},
	}

	id, err := s.docker.ContainerCreate(context.TODO(), containerConfig, hostConfig, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	if err := s.docker.ContainerStart(context.TODO(), id); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	return id, nil
}

// StopNamedContainer stops a named sing-box container.
func (s *Service) StopNamedContainer(name string) error {
	containerName := "singbox-" + name
	state, err := s.docker.GetContainerState(context.TODO(), containerName)
	if err != nil {
		return err
	}
	if state == "" {
		return nil
	}
	timeout := 10
	return s.docker.ContainerStop(context.TODO(), containerName, &timeout)
}

// NamedContainerStatus returns whether a named container is running and its ID.
func (s *Service) NamedContainerStatus(name string) (running bool, containerID string) {
	containerName := "singbox-" + name
	state, err := s.docker.GetContainerState(context.TODO(), containerName)
	if err != nil || state == "" {
		return false, ""
	}
	return state == "running", state
}

// NamedContainerLogs returns the last 100 log lines from a named container.
func (s *Service) NamedContainerLogs(name string) string {
	containerName := "singbox-" + name
	logs, err := s.docker.ContainerLogs(context.TODO(), containerName, "100")
	if err != nil {
		return fmt.Sprintf("Error getting logs: %v", err)
	}
	return logs
}

// CheckNamedConfig validates a named configuration's JSON syntax.
func (s *Service) CheckNamedConfig(name string) (valid bool, output string) {
	configPath := s.getNamedConfigPath(name)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, fmt.Sprintf("Config not found: %v", err)
	}
	var js interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		return false, fmt.Sprintf("Invalid JSON: %v", err)
	}
	return true, "Config is valid JSON"
}

// ListAllContainers returns all containers with the "singbox" prefix.
func (s *Service) ListAllContainers() ([]docker.ContainerInfo, error) {
	return s.docker.ListContainers(context.TODO(), "singbox")
}
