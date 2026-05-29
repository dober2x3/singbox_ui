package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// NodeMapping stores outbound tag -> node name mapping, used for fallback matching during subscription auto-update
// File is stored in data directory, not mixed with sing-box config

var nodeMappingMu sync.RWMutex

func getNodeMappingFilePath() string {
	baseDir := os.Getenv("DATA_DIR")
	if baseDir == "" {
		baseDir, _ = os.Getwd()
	}
	return filepath.Join(baseDir, "node-mapping.json")
}

// LoadNodeMapping loads tag -> nodeName mapping
func LoadNodeMapping() map[string]string {
	nodeMappingMu.RLock()
	defer nodeMappingMu.RUnlock()

	data, err := os.ReadFile(getNodeMappingFilePath())
	if err != nil {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]string{}
	}
	return m
}

// SaveNodeMapping saves tag -> nodeName mapping
func SaveNodeMapping(m map[string]string) error {
	nodeMappingMu.Lock()
	defer nodeMappingMu.Unlock()

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getNodeMappingFilePath(), data, 0644)
}

// UpsertNodeMapping updates or inserts a single tag -> nodeName record
func UpsertNodeMapping(tag, nodeName string) {
	m := LoadNodeMapping()
	m[tag] = nodeName
	_ = SaveNodeMapping(m)
}
