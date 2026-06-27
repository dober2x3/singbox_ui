// Package config provides application configuration loaded from environment variables
// and filesystem state. It manages data directories, listen addresses, and host path
// resolution for container environments.
package config

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds HTTP server and path configuration.
type ServerConfig struct {
	ListenAddr      string `yaml:"listen_addr"`
	DataDir         string `yaml:"data_dir"`
	HostDataDir     string `yaml:"host_data_dir"`
	SingboxBinPath  string `yaml:"singbox_bin_path"`
	ServeDashboard  bool   `yaml:"serve_dashboard"`
}

// AppConfig is the top-level application configuration.
// Domain-specific sections (Prober, Speedtest, Scheduler, Subscription) are stored
// as raw yaml.Node and parsed by each domain's ParseConfig function.
type AppConfig struct {
	Server       ServerConfig `yaml:"server"`
	Prober       yaml.Node    `yaml:"prober"`
	Speedtest    yaml.Node    `yaml:"speedtest"`
	Scheduler    yaml.Node    `yaml:"scheduler"`
	Subscription yaml.Node    `yaml:"subscription"`
}

// defaultDataDir returns the default data directory: $HOME/singbox_ui.
// Falls back to working directory if home dir cannot be determined.
func defaultDataDir() string {
	if u, err := user.Current(); err == nil {
		return filepath.Join(u.HomeDir, "singbox_ui")
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// Load reads a YAML config file and returns an AppConfig with defaults applied.
func Load(path string) (*AppConfig, error) {
	var cfg AppConfig
	cfg.Server.ListenAddr = "127.0.0.1:7000"

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if cfg.Server.ListenAddr == "" {
		cfg.Server.ListenAddr = "127.0.0.1:7000"
	}

	return &cfg, nil
}

// Init bootstraps configuration. It resolves the data directory and config file path.
// If configPath is non-empty, it is used directly.
// Otherwise DATA_DIR env var is used, or $HOME/singbox_ui as fallback.
func Init(configPath string) (*AppConfig, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = defaultDataDir()
	}

	path := configPath
	if path == "" {
		path = filepath.Join(dataDir, "config.yaml")
	}

	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	if cfg.Server.DataDir == "" {
		cfg.Server.DataDir = dataDir
	}

	// Backwards compatibility: if config file did not set listen_addr/host_data_dir,
	// check old env vars.
	if cfg.Server.ListenAddr == "127.0.0.1:7000" {
		if envAddr := os.Getenv("LISTEN_ADDR"); envAddr != "" {
			cfg.Server.ListenAddr = envAddr
		}
	}
	if cfg.Server.HostDataDir == "" {
		if envHost := os.Getenv("HOST_DATA_DIR"); envHost != "" {
			cfg.Server.HostDataDir = envHost
		}
	}

	singboxDir := filepath.Join(cfg.Server.DataDir, "singbox")
	if err := os.MkdirAll(singboxDir, 0755); err != nil {
		log.Printf("Warning: failed to create singbox directory: %v", err)
	}

	return cfg, nil
}

// GetDataDir returns the application data directory path.
func (c *AppConfig) GetDataDir() string {
	return c.Server.DataDir
}

// GetListenAddr returns the HTTP server listen address (host:port).
func (c *AppConfig) GetListenAddr() string {
	return c.Server.ListenAddr
}

// GetSingboxDir returns the sing-box configuration directory path.
func (c *AppConfig) GetSingboxDir() string {
	return filepath.Join(c.Server.DataDir, "singbox")
}

// GetSingboxBinPath returns the path to the sing-box binary for native mode.
func (c *AppConfig) GetSingboxBinPath() string {
	return c.Server.SingboxBinPath
}

// ResolveHostConfigDir converts a container-internal path under DATA_DIR to the
// corresponding host path using HOST_DATA_DIR.
func (c *AppConfig) ResolveHostConfigDir(containerPath string) (string, error) {
	if c.Server.HostDataDir == "" {
		return "", fmt.Errorf("HOST_DATA_DIR environment variable is not set")
	}
	rel, err := filepath.Rel(c.Server.DataDir, containerPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is not under DATA_DIR %s", containerPath, c.Server.DataDir)
	}
	return filepath.Join(c.Server.HostDataDir, rel), nil
}
