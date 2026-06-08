// Package singbox provides services for managing sing-box configuration and containers.
package singbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"singbox-config-service/internal/pkg/config"
)

// Service provides business logic for sing-box configuration and instance management.
type Service struct {
	runtime Runtime
	cfg     *config.AppConfig
}

// NewService creates a new Service with the given Runtime and Config.
func NewService(runtime Runtime, cfg *config.AppConfig) *Service {
	return &Service{
		runtime: runtime,
		cfg:     cfg,
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

// RunContainer creates and starts a sing-box instance (Docker container or native process).
func (s *Service) RunContainer() (string, error) {
	configPath := filepath.Join(s.cfg.GetSingboxDir(), "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config file not found, please save config first")
	}

	return s.runtime.Start(context.TODO(), "default", configPath)
}

// StopContainer stops the sing-box instance.
func (s *Service) StopContainer() error {
	timeout := 10
	return s.runtime.Stop(context.TODO(), "default", &timeout)
}

// ContainerStatus returns whether the sing-box instance is running and its identifier.
func (s *Service) ContainerStatus() (running bool, containerID string) {
	running, id, err := s.runtime.Status(context.TODO(), "default")
	if err != nil {
		return false, ""
	}
	return running, id
}

// ContainerLogs returns the last log lines from the sing-box instance.
func (s *Service) ContainerLogs() string {
	logs, err := s.runtime.Logs(context.TODO(), "default", "100")
	if err != nil {
		return fmt.Sprintf("Error getting logs: %v", err)
	}
	return logs
}

// EnsureImage is a no-op — image/process readiness is handled by the Runtime implementation.
func (s *Service) EnsureImage() error {
	return nil
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

// RunNamedContainer creates and starts a named sing-box instance.
func (s *Service) RunNamedContainer(name string) (string, error) {
	configPath := s.getNamedConfigPath(name)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config for '%s' not found", name)
	}

	running, _, err := s.runtime.Status(context.TODO(), name)
	if err == nil && running {
		return "", fmt.Errorf("container '%s' is already running", name)
	}

	return s.runtime.Start(context.TODO(), name, configPath)
}

// StopNamedContainer stops a named sing-box instance.
func (s *Service) StopNamedContainer(name string) error {
	timeout := 10
	return s.runtime.Stop(context.TODO(), name, &timeout)
}

// NamedContainerStatus returns whether a named instance is running and its identifier.
func (s *Service) NamedContainerStatus(name string) (running bool, containerID string) {
	running, id, err := s.runtime.Status(context.TODO(), name)
	if err != nil {
		return false, ""
	}
	return running, id
}

// NamedContainerLogs returns the last log lines from a named instance.
func (s *Service) NamedContainerLogs(name string) string {
	logs, err := s.runtime.Logs(context.TODO(), name, "100")
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

// ListAllContainers returns all instances with their status.
func (s *Service) ListAllContainers() ([]InstanceInfo, error) {
	return s.runtime.List(context.TODO())
}
