package speedtest

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseConfig_nilNode(t *testing.T) {
	cfg, err := ParseConfig(nil)
	if err != nil {
		t.Fatalf("ParseConfig(nil) error = %v", err)
	}
	def := DefaultConfig()
	if cfg != def {
		t.Errorf("ParseConfig(nil) = %+v, want %+v", cfg, def)
	}
}

func TestParseConfig_zeroNode(t *testing.T) {
	var node yaml.Node
	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(zero node) error = %v", err)
	}
	def := DefaultConfig()
	if cfg != def {
		t.Errorf("ParseConfig(zero node) = %+v, want %+v", cfg, def)
	}
}

func TestParseConfig_partial(t *testing.T) {
	yamlContent := []byte(`duration: 30`)
	var node yaml.Node
	if err := yaml.Unmarshal(yamlContent, &node); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(partial) error = %v", err)
	}
	if cfg.Duration != 30 {
		t.Errorf("Duration = %d, want 30", cfg.Duration)
	}
	def := DefaultConfig()
	if cfg.LatencyURL != def.LatencyURL {
		t.Errorf("LatencyURL = %q, want default %q", cfg.LatencyURL, def.LatencyURL)
	}
	if cfg.DownloadURL != def.DownloadURL {
		t.Errorf("DownloadURL = %q, want default %q", cfg.DownloadURL, def.DownloadURL)
	}
}

func TestParseConfig_full(t *testing.T) {
	yamlContent := []byte(`
latency_url: "http://example.com/latency"
download_url: "http://example.com/download"
duration: 20
`)
	var node yaml.Node
	if err := yaml.Unmarshal(yamlContent, &node); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(full) error = %v", err)
	}
	if cfg.LatencyURL != "http://example.com/latency" {
		t.Errorf("LatencyURL = %q, want http://example.com/latency", cfg.LatencyURL)
	}
	if cfg.DownloadURL != "http://example.com/download" {
		t.Errorf("DownloadURL = %q, want http://example.com/download", cfg.DownloadURL)
	}
	if cfg.Duration != 20 {
		t.Errorf("Duration = %d, want 20", cfg.Duration)
	}
}

func TestParseConfig_invalidYAML(t *testing.T) {
	var node yaml.Node
	node.Kind = yaml.ScalarNode
	node.Value = "not-a-mapping"

	_, err := ParseConfig(&node)
	if err == nil {
		t.Error("ParseConfig(scalar) expected error, got nil")
	}
}
