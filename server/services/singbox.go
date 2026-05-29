package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	baseDir    string
	singboxDir string
)

// DockerService instance
var (
	dockerService *DockerService
	dockerMutex   sync.RWMutex
)

// init initializes path variables
func init() {
	// Prefer data directory from environment variable
	dataDir := os.Getenv("DATA_DIR")
	if dataDir != "" {
		baseDir = dataDir
		singboxDir = filepath.Join(baseDir, "singbox")
		log.Printf("Using DATA_DIR: %s", baseDir)
		return
	}

	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Printf("Warning: Failed to get working directory: %v", err)
		baseDir = "."
	} else {
		// If working directory contains go.mod, it is the server directory
		if _, err := os.Stat(filepath.Join(workDir, "go.mod")); err == nil {
			baseDir = workDir
		} else if _, err := os.Stat(filepath.Join(workDir, "server", "go.mod")); err == nil {
			// If it's the project root, use server subdirectory
			baseDir = filepath.Join(workDir, "server")
		} else {
			// Default to current working directory
			baseDir = workDir
		}
	}

	singboxDir = filepath.Join(baseDir, "singbox")
}

// GetSingboxDir gets the sing-box config directory
func GetSingboxDir() string {
	return singboxDir
}

// InitDockerService initializes the Docker service
func InitDockerService() error {
	dockerMutex.Lock()
	defer dockerMutex.Unlock()

	if dockerService != nil {
		return nil
	}

	var err error
	dockerService, err = NewDockerService()
	if err != nil {
		return fmt.Errorf("failed to initialize docker service: %v", err)
	}

	log.Println("Docker service initialized successfully")
	return nil
}

// GetDockerService gets the Docker service instance
func GetDockerService() *DockerService {
	dockerMutex.RLock()
	defer dockerMutex.RUnlock()
	return dockerService
}

// EnsureSingboxImage ensures the sing-box image exists
func EnsureSingboxImage() error {
	ds := GetDockerService()
	if ds == nil {
		return fmt.Errorf("docker service not initialized")
	}
	return ds.EnsureImage()
}

// RunSingboxContainer starts the sing-box container
func RunSingboxContainer() (string, error) {
	ds := GetDockerService()
	if ds == nil {
		return "", fmt.Errorf("docker service not initialized")
	}

	// Ensure config file exists
	configPath := filepath.Join(singboxDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config file not found: %s", configPath)
	}

	// Create and start container
	containerID, err := ds.CreateAndStartContainer(singboxDir)
	if err != nil {
		return "", err
	}

	log.Printf("sing-box container started: %s", containerID[:12])
	return containerID, nil
}

// StopSingboxContainer stops the sing-box container
func StopSingboxContainer() error {
	ds := GetDockerService()
	if ds == nil {
		return fmt.Errorf("docker service not initialized")
	}

	if err := ds.StopContainer(); err != nil {
		return err
	}

	if err := ds.RemoveContainer(); err != nil {
		return err
	}

	log.Println("sing-box container stopped and removed")
	return nil
}

// CheckContainerRunning checks if the container is running
func CheckContainerRunning() (bool, string) {
	ds := GetDockerService()
	if ds == nil {
		return false, ""
	}

	running, containerID, err := ds.GetContainerStatus()
	if err != nil {
		log.Printf("Failed to check container status: %v", err)
		return false, ""
	}

	return running, containerID
}

// GetContainerLogs gets the container logs
func GetContainerLogs() string {
	ds := GetDockerService()
	if ds == nil {
		return "Docker service not initialized"
	}

	logs, err := ds.GetContainerLogs("200")
	if err != nil {
		log.Printf("Failed to get container logs: %v", err)
		return fmt.Sprintf("Failed to get logs: %v", err)
	}

	return logs
}

// GetSingBoxVersion gets the sing-box version (from container)
func GetSingBoxVersion() (string, error) {
	ds := GetDockerService()
	if ds == nil {
		return "", fmt.Errorf("docker service not initialized")
	}

	return ds.GetSingBoxVersion()
}

// SaveConfig saves the config file and auto-updates the tag->nodeName side mapping
func SaveConfig(configData []byte) (string, error) {
	// Ensure singbox directory exists
	if err := os.MkdirAll(singboxDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create singbox directory: %w", err)
	}

	configPath := filepath.Join(singboxDir, "config.json")

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return "", fmt.Errorf("failed to save config: %w", err)
	}

	// Async update tag->nodeName mapping (does not block save flow)
	go rebuildNodeMapping(configData)

	return configPath, nil
}

// rebuildNodeMapping extracts tags from config outbounds, matches with subscription nodes and updates mapping file
func rebuildNodeMapping(configData []byte) {
	var cfg struct {
		Outbounds []map[string]interface{} `json:"outbounds"`
	}
	if err := json.Unmarshal(configData, &cfg); err != nil {
		return
	}

	// Load subscription nodes, build server:port:type -> name index
	subData, err := LoadSubscriptions()
	if err != nil {
		return
	}
	type key struct{ server string; port int; proto string }
	nodeNameByKey := make(map[key]string)
	for _, sub := range subData.Subscriptions {
		for _, n := range sub.Nodes {
			nodeNameByKey[key{n.Address, n.Port, n.Protocol}] = n.Name
		}
	}

	mapping := LoadNodeMapping()
	for _, ob := range cfg.Outbounds {
		obType, _ := ob["type"].(string)
		if obType == "direct" || obType == "block" || obType == "dns" || obType == "urltest" || obType == "selector" {
			continue
		}
		tag, _ := ob["tag"].(string)
		if tag == "" {
			continue
		}
		server, _ := ob["server"].(string)
		portFloat, _ := ob["server_port"].(float64)
		port := int(portFloat)
		if name, ok := nodeNameByKey[key{server, port, obType}]; ok {
			mapping[tag] = name
		}
	}
	_ = SaveNodeMapping(mapping)
}

// GetConfig gets the config file
func GetConfig() ([]byte, error) {
	configPath := filepath.Join(singboxDir, "config.json")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found")
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return data, nil
}

// CloseDockerService closes the Docker service
func CloseDockerService() error {
	dockerMutex.Lock()
	defer dockerMutex.Unlock()

	if dockerService != nil {
		err := dockerService.Close()
		dockerService = nil
		return err
	}
	return nil
}

// ========== Multi-config multi-container management ==========

// getNamedConfigDir gets the named config directory
func getNamedConfigDir(name string) string {
	return filepath.Join(singboxDir, "configs", name)
}

// RunNamedContainer starts the named config container
func RunNamedContainer(name string) (string, error) {
	ds := GetDockerService()
	if ds == nil {
		return "", fmt.Errorf("docker service not initialized")
	}

	// Ensure config directory exists
	configDir := getNamedConfigDir(name)
	configPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config file not found: %s", configPath)
	}

	// Create and start container
	containerID, err := ds.CreateAndStartNamedContainer(name, configDir)
	if err != nil {
		return "", err
	}

	log.Printf("Named sing-box container started for config %s: %s", name, containerID[:12])
	return containerID, nil
}

// StopNamedContainer stops the named config container
func StopNamedContainer(name string) error {
	ds := GetDockerService()
	if ds == nil {
		return fmt.Errorf("docker service not initialized")
	}

	if err := ds.StopNamedContainer(name); err != nil {
		return err
	}

	if err := ds.RemoveNamedContainer(name); err != nil {
		return err
	}

	log.Printf("Named sing-box container stopped for config: %s", name)
	return nil
}

// GetNamedContainerStatus gets the named container status
func GetNamedContainerStatus(name string) (bool, string) {
	ds := GetDockerService()
	if ds == nil {
		return false, ""
	}

	running, containerID, err := ds.GetNamedContainerStatus(name)
	if err != nil {
		log.Printf("Failed to check named container status: %v", err)
		return false, ""
	}

	return running, containerID
}

// GetNamedContainerLogs gets the named container logs
func GetNamedContainerLogs(name string) string {
	ds := GetDockerService()
	if ds == nil {
		return "Docker service not initialized"
	}

	logs, err := ds.GetNamedContainerLogs(name, "200")
	if err != nil {
		log.Printf("Failed to get named container logs: %v", err)
		return fmt.Sprintf("Failed to get logs: %v", err)
	}

	return logs
}

// ListAllContainers lists all sing-box containers
func ListAllContainers() ([]ContainerInfo, error) {
	ds := GetDockerService()
	if ds == nil {
		return nil, fmt.Errorf("docker service not initialized")
	}

	return ds.ListAllSingboxContainers()
}

// CheckNamedConfig validates the named config
func CheckNamedConfig(name string) (bool, string, error) {
	ds := GetDockerService()
	if ds == nil {
		return false, "", fmt.Errorf("docker service not initialized")
	}

	configDir := getNamedConfigDir(name)
	configPath := filepath.Join(configDir, "config.json")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, "", fmt.Errorf("config file not found for instance: %s", name)
	}

	return ds.CheckNamedConfig(name, configDir)
}

// SaveNamedConfigWithDir saves config to named directory (for multi-container scenarios)
func SaveNamedConfigWithDir(name string, configData []byte) error {
	configDir := getNamedConfigDir(name)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	log.Printf("Named config saved: %s", configPath)
	return nil
}

// LoadNamedConfigFromDir loads config from named directory
func LoadNamedConfigFromDir(name string) ([]byte, error) {
	configPath := filepath.Join(getNamedConfigDir(name), "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config not found: %s", name)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return data, nil
}

// DeleteNamedConfigWithDir deletes the named config directory
func DeleteNamedConfigWithDir(name string) error {
	// First stop the container (if running)
	running, _ := GetNamedContainerStatus(name)
	if running {
		if err := StopNamedContainer(name); err != nil {
			log.Printf("Warning: failed to stop container before delete: %v", err)
		}
	}

	configDir := getNamedConfigDir(name)
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("failed to delete config directory: %w", err)
	}

	log.Printf("Named config deleted: %s", name)
	return nil
}

// ListNamedConfigs lists all named configs and their container status
func ListNamedConfigs() ([]NamedConfigInfo, error) {
	configsDir := filepath.Join(singboxDir, "configs")

	// Ensure directory exists
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create configs directory: %w", err)
	}

	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read configs directory: %w", err)
	}

	var configs []NamedConfigInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		configPath := filepath.Join(configsDir, name, "config.json")

		// Check if config file exists
		info, err := os.Stat(configPath)
		if os.IsNotExist(err) {
			continue
		}

		// Get container status
		running, containerID := GetNamedContainerStatus(name)

		configs = append(configs, NamedConfigInfo{
			Name:        name,
			CreatedAt:   info.ModTime().Unix(),
			Size:        info.Size(),
			Running:     running,
			ContainerID: containerID,
		})
	}

	return configs, nil
}

// NamedConfigInfo named config info (includes container status)
type NamedConfigInfo struct {
	Name        string `json:"name"`
	CreatedAt   int64  `json:"created_at"`
	Size        int64  `json:"size"`
	Running     bool   `json:"running"`
	ContainerID string `json:"container_id,omitempty"`
}
