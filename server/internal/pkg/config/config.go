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
)

// Config holds application configuration values derived from environment variables
// and the runtime environment. All fields are unexported; access via getter methods.
type Config struct {
	dataDir     string
	hostDataDir string
	listenAddr  string
	singboxDir  string
}

// Init initializes a Config from environment variables. It reads DATA_DIR, LISTEN_ADDR,
// and HOST_DATA_DIR, falling back to sensible defaults where unset. The singbox
// subdirectory under DATA_DIR is created if it does not exist.
func Init() (*Config, error) {
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

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "127.0.0.1:7000"
	}
	hostDataDir := os.Getenv("HOST_DATA_DIR")

	cfg := &Config{
		dataDir:     dataDir,
		hostDataDir: hostDataDir,
		listenAddr:  listenAddr,
		singboxDir:  filepath.Join(dataDir, "singbox"),
	}

	if err := os.MkdirAll(cfg.singboxDir, 0755); err != nil {
		log.Printf("Warning: failed to create singbox directory: %v", err)
	}

	return cfg, nil
}

// GetDataDir returns the application data directory path.
func (c *Config) GetDataDir() string {
	return c.dataDir
}

// GetSingboxDir returns the sing-box configuration directory path.
func (c *Config) GetSingboxDir() string {
	return c.singboxDir
}

// GetListenAddr returns the HTTP server listen address (host:port).
func (c *Config) GetListenAddr() string {
	return c.listenAddr
}

// ResolveHostConfigDir converts a container-internal path under DATA_DIR to the
// corresponding host path using HOST_DATA_DIR. Returns an error if HOST_DATA_DIR
// is not set or if the path falls outside DATA_DIR.
func (c *Config) ResolveHostConfigDir(containerPath string) (string, error) {
	if c.hostDataDir == "" {
		return "", fmt.Errorf("HOST_DATA_DIR environment variable is not set")
	}

	rel, err := filepath.Rel(c.dataDir, containerPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is not under DATA_DIR %s", containerPath, c.dataDir)
	}
	return filepath.Join(c.hostDataDir, rel), nil
}
