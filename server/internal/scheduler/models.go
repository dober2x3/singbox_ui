package scheduler

import (
	"gopkg.in/yaml.v3"
)

// Config holds configuration parameters for the scheduler.
type Config struct {
	Interval int `json:"interval" yaml:"interval" example:"60"` // seconds
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{Interval: 60}
}

// ParseConfig parses a yaml.Node into a Config, applying defaults.
func ParseConfig(node *yaml.Node) (Config, error) {
	cfg := DefaultConfig()
	if node == nil || node.Kind == 0 {
		return cfg, nil
	}
	if err := node.Decode(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.Interval == 0 {
		cfg.Interval = DefaultConfig().Interval
	}
	return cfg, nil
}
