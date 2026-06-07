// Package config provides application configuration loaded from environment variables
// and filesystem state. It manages data directories, listen addresses, and host path
// resolution for container environments.
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"singbox-config-service/internal/prober"
	"singbox-config-service/internal/scheduler"
	"singbox-config-service/internal/subscription"

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

// SpeedtestConfig holds configuration for the speed test service.
// Defined inline here because speedtest imports config (would create a cycle).
type SpeedtestConfig struct {
	LatencyURL  string `yaml:"latency_url"`
	DownloadURL string `yaml:"download_url"`
	Duration    int    `yaml:"duration"`
}

// defaultSpeedtestConfig returns a SpeedtestConfig with sensible defaults.
func defaultSpeedtestConfig() SpeedtestConfig {
	return SpeedtestConfig{
		LatencyURL:  "http://www.gstatic.com/generate_204",
		DownloadURL: "https://speed.cloudflare.com/__down?bytes=10000000",
		Duration:    10,
	}
}

// AppConfig is the top-level application configuration.
type AppConfig struct {
	Server       ServerConfig         `yaml:"server"`
	Prober       prober.Config        `yaml:"prober"`
	Speedtest    SpeedtestConfig      `yaml:"speedtest"`
	Scheduler    scheduler.Config     `yaml:"scheduler"`
	Subscription subscription.Config  `yaml:"subscription"`
}

// defaultAppConfig returns an AppConfig with all defaults set.
func defaultAppConfig() AppConfig {
	return AppConfig{
		Server: ServerConfig{
			ListenAddr: "127.0.0.1:7000",
		},
		Prober:       prober.DefaultConfig(),
		Speedtest:    defaultSpeedtestConfig(),
		Scheduler:    scheduler.DefaultConfig(),
		Subscription: subscription.DefaultConfig(),
	}
}

// Load reads a YAML config file and merges defaults for zero-valued fields.
func Load(path string) (*AppConfig, error) {
	cfg := defaultAppConfig()

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

	cfg = mergeDefaults(cfg)

	return &cfg, nil
}

// mergeDefaults fills zero-valued fields with defaults.
func mergeDefaults(cfg AppConfig) AppConfig {
	def := defaultAppConfig()

	if cfg.Server.ListenAddr == "" {
		cfg.Server.ListenAddr = def.Server.ListenAddr
	}
	if cfg.Prober.Interval == 0 {
		cfg.Prober.Interval = def.Prober.Interval
	}
	if cfg.Prober.Timeout == 0 {
		cfg.Prober.Timeout = def.Prober.Timeout
	}
	if cfg.Prober.Concurrent == 0 {
		cfg.Prober.Concurrent = def.Prober.Concurrent
	}
	if cfg.Prober.MaxResults == 0 {
		cfg.Prober.MaxResults = def.Prober.MaxResults
	}
	if cfg.Prober.MaxRetries == 0 {
		cfg.Prober.MaxRetries = def.Prober.MaxRetries
	}
	if cfg.Speedtest.LatencyURL == "" {
		cfg.Speedtest.LatencyURL = def.Speedtest.LatencyURL
	}
	if cfg.Speedtest.DownloadURL == "" {
		cfg.Speedtest.DownloadURL = def.Speedtest.DownloadURL
	}
	if cfg.Speedtest.Duration == 0 {
		cfg.Speedtest.Duration = def.Speedtest.Duration
	}
	if cfg.Scheduler.Interval == 0 {
		cfg.Scheduler.Interval = def.Scheduler.Interval
	}
	return cfg
}

// Init bootstraps configuration. It resolves the data directory and config file path.
// If configPath is non-empty, it is used directly. Otherwise DATA_DIR/config.yaml is tried.
func Init(configPath string) (*AppConfig, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		workDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine working directory: %w", err)
		}
		if _, err := os.Stat(filepath.Join(workDir, "go.mod")); err == nil {
			dataDir = workDir
		} else if _, err := os.Stat(filepath.Join(workDir, "server", "go.mod")); err == nil {
			dataDir = filepath.Join(workDir, "server")
		} else {
			dataDir = workDir
		}
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
