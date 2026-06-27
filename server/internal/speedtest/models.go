package speedtest

import (
	"gopkg.in/yaml.v3"
)

// SpeedTestState represents the current state of a speed test.
type SpeedTestState struct {
	Running       bool    `json:"running" example:"true"`
	Tag           string  `json:"tag,omitempty" example:"my-proxy"`
	Status        string  `json:"status,omitempty" example:"testing..."`
	Progress      int     `json:"progress,omitempty" example:"50"`
	DownloadSpeed float64 `json:"download_speed,omitempty" example:"15.5"`
	LatencyMs     int64   `json:"latency_ms,omitempty" example:"120"`
	Error         string  `json:"error,omitempty" example:"timeout"`
}

// Config holds configuration parameters for the speed test service.
type Config struct {
	LatencyURL  string `json:"latency_url" yaml:"latency_url" example:"http://www.gstatic.com/generate_204"`
	DownloadURL string `json:"download_url" yaml:"download_url" example:"https://speed.cloudflare.com/__down?bytes=10000000"`
	Duration    int    `json:"duration" yaml:"duration" example:"10"` // seconds
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		LatencyURL:  "http://www.gstatic.com/generate_204",
		DownloadURL: "https://speed.cloudflare.com/__down?bytes=10000000",
		Duration:    10,
	}
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
	def := DefaultConfig()
	if cfg.LatencyURL == "" {
		cfg.LatencyURL = def.LatencyURL
	}
	if cfg.DownloadURL == "" {
		cfg.DownloadURL = def.DownloadURL
	}
	if cfg.Duration == 0 {
		cfg.Duration = def.Duration
	}
	return cfg, nil
}
