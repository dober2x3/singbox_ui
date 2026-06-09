package resourcecheck

import (
	"os"

	"gopkg.in/yaml.v3"
)

// ResourceConfig defines a single resource to check, loaded from resources.yaml.
type ResourceConfig struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	Type string `yaml:"type"`       // "http" | "tcp"
	Port int    `yaml:"port,omitempty"`
}

// ResourceConfigFile is the root YAML structure.
type ResourceConfigFile struct {
	Resources []ResourceConfig `yaml:"resources"`
}

// LoadResources reads resources from a YAML file. Returns empty slice if file missing.
func LoadResources(path string) ([]ResourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg ResourceConfigFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.Resources, nil
}

// CheckResult represents a single resource check result persisted in SQLite.
type CheckResult struct {
	ID        int64  `json:"id" db:"id"`
	Resource  string `json:"resource" db:"resource"`
	Tag       string `json:"tag" db:"tag"`
	Status    string `json:"status" db:"status"`    // "ok" | "timeout" | "error"
	LatencyMs int64  `json:"latency_ms" db:"latency_ms"`
	HTTPCode  int    `json:"http_code,omitempty" db:"http_code"`
	Error     string `json:"error,omitempty" db:"error"`
	CheckedAt string `json:"checked_at" db:"checked_at"` // ISO 8601
}

// CheckStatus tracks progress of a running check operation.
type CheckStatus struct {
	Running         bool   `json:"running"`
	Tag             string `json:"tag,omitempty"`
	Resource        string `json:"resource,omitempty"`
	Progress        int    `json:"progress,omitempty"`
	TotalNodes      int    `json:"total_nodes,omitempty"`
	CompletedNodes  int    `json:"completed_nodes,omitempty"`
	TotalChecks     int    `json:"total_checks,omitempty"`
	CompletedChecks int    `json:"completed_checks,omitempty"`
	Status          string `json:"status,omitempty"` // "idle" | "running" | "completed"
}

// RunRequest is the body for POST /api/resourcecheck/run.
type RunRequest struct {
	Tag            string `json:"tag,omitempty"`
	SubscriptionID string `json:"subscription_id,omitempty"`
}

// ScheduleRequest is the body for POST /api/resourcecheck/schedule.
type ScheduleRequest struct {
	IntervalSec int `json:"interval_sec"`
}
