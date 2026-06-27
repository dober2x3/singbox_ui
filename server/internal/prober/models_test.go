package prober

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
	var node yaml.Node // Kind == 0
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
	yamlContent := []byte(`interval: 45`)
	var node yaml.Node
	if err := yaml.Unmarshal(yamlContent, &node); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(partial) error = %v", err)
	}

	if cfg.Interval != 45 {
		t.Errorf("Interval = %d, want 45", cfg.Interval)
	}
	// Other fields should be defaults
	def := DefaultConfig()
	if cfg.Timeout != def.Timeout {
		t.Errorf("Timeout = %d, want default %d", cfg.Timeout, def.Timeout)
	}
	if cfg.Concurrent != def.Concurrent {
		t.Errorf("Concurrent = %d, want default %d", cfg.Concurrent, def.Concurrent)
	}
}

func TestParseConfig_full(t *testing.T) {
	yamlContent := []byte(`
interval: 60
timeout: 3000
concurrent: 10
max_results: 200
max_retries: 3
`)
	var node yaml.Node
	if err := yaml.Unmarshal(yamlContent, &node); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(full) error = %v", err)
	}

	if cfg.Interval != 60 {
		t.Errorf("Interval = %d, want 60", cfg.Interval)
	}
	if cfg.Timeout != 3000 {
		t.Errorf("Timeout = %d, want 3000", cfg.Timeout)
	}
	if cfg.Concurrent != 10 {
		t.Errorf("Concurrent = %d, want 10", cfg.Concurrent)
	}
	if cfg.MaxResults != 200 {
		t.Errorf("MaxResults = %d, want 200", cfg.MaxResults)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
}

func TestParseConfig_invalidYAML(t *testing.T) {
	// Create a node that will produce a decode error (scalar instead of mapping)
	var node yaml.Node
	node.Kind = yaml.ScalarNode
	node.Value = "not-a-mapping"

	_, err := ParseConfig(&node)
	if err == nil {
		t.Error("ParseConfig(scalar) expected error, got nil")
	}
}
