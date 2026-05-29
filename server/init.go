package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"singbox-config-service/services"
)

var (
	baseDir          string
	singboxDir       string
	singboxConfigFile string
)

// init initialize path variables
func init() {
	// use the data directory specified by environment variable first
	dataDir := os.Getenv("DATA_DIR")
	if dataDir != "" {
		baseDir = dataDir
		singboxDir = filepath.Join(baseDir, "singbox")
		singboxConfigFile = filepath.Join(singboxDir, "config.json")
		log.Printf("Using DATA_DIR: %s", baseDir)
		log.Printf("sing-box directory: %s", singboxDir)
		return
	}

	// get working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// if working directory contains go.mod, it's the server directory
	if _, err := os.Stat(filepath.Join(workDir, "go.mod")); err == nil {
		baseDir = workDir
	} else if _, err := os.Stat(filepath.Join(workDir, "server", "go.mod")); err == nil {
		// if it's the project root, use the server subdirectory
		baseDir = filepath.Join(workDir, "server")
	} else {
		// default to current working directory
		baseDir = workDir
	}

	singboxDir = filepath.Join(baseDir, "singbox")
	singboxConfigFile = filepath.Join(singboxDir, "config.json")

	log.Printf("Base directory: %s", baseDir)
	log.Printf("sing-box directory: %s", singboxDir)
}

// Initialize initialize sing-box Docker environment
func Initialize() error {
	log.Println("Initializing sing-box Docker environment...")

	// create singbox directory
	if err := os.MkdirAll(singboxDir, 0755); err != nil {
		return fmt.Errorf("failed to create singbox directory: %v", err)
	}

	// initialize Docker service
	if err := services.InitDockerService(); err != nil {
		return fmt.Errorf("failed to initialize docker service: %v", err)
	}

	// pull sing-box image in background
	go func() {
		log.Println("Pulling sing-box image in background...")
		if err := services.EnsureSingboxImage(); err != nil {
			log.Printf("Warning: Failed to pull sing-box image: %v", err)
		} else {
			log.Println("sing-box image is ready")
		}
	}()

	// check config file
	if _, err := os.Stat(singboxConfigFile); os.IsNotExist(err) {
		log.Println("Creating default sing-box config...")
		if err := createDefaultConfig(); err != nil {
			return fmt.Errorf("failed to create default config: %v", err)
		}
	}

	// initialize prober
	if err := services.InitProber(); err != nil {
		log.Printf("Warning: Failed to initialize prober: %v", err)
	} else {
		// load saved nodes and start prober
		prober := services.GetProber()
		if prober != nil {
			if err := prober.LoadNodesFromFile(); err != nil {
				log.Printf("Warning: Failed to load prober nodes: %v", err)
			}
			// if there are nodes, auto-start prober
			if len(prober.GetAllResults()) > 0 {
				prober.Start()
				log.Println("Prober started with saved nodes")
			}
		}
	}

	// start subscription auto-update scheduler
	services.StartAutoUpdateScheduler()

	log.Println("sing-box Docker environment initialized successfully")
	return nil
}

// createDefaultConfig create default config file (sing-box format)
func createDefaultConfig() error {
	defaultConfig := map[string]interface{}{
		"log": map[string]interface{}{
			"level":     "warn",
			"timestamp": true,
		},
		"dns": map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{
					"tag":    "local_dns",
					"type":   "udp",
					"server": "223.5.5.5",
				},
				map[string]interface{}{
					"tag":    "remote_dns",
					"type":   "udp",
					"server": "8.8.8.8",
				},
			},
			"final":             "remote_dns",
			"independent_cache": true,
		},
		"inbounds":  []interface{}{},
		"outbounds": []interface{}{
			map[string]interface{}{
				"type": "direct",
				"tag":  "direct",
			},
			map[string]interface{}{
				"type": "block",
				"tag":  "block",
			},
		},
		"route": map[string]interface{}{
			"rules": []interface{}{},
			"final": "direct",
		},
	}

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(singboxConfigFile, data, 0644)
}
