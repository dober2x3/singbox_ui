package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	dataDir     string
	hostDataDir string
	listenAddr  string
	singboxDir  string
}

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

func (c *Config) GetDataDir() string {
	return c.dataDir
}

func (c *Config) GetSingboxDir() string {
	return c.singboxDir
}

func (c *Config) GetListenAddr() string {
	return c.listenAddr
}

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
