package scheduler

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseConfig_nilNode(t *testing.T) {
	cfg, err := ParseConfig(nil)
	if err != nil {
		t.Fatalf("ParseConfig(nil) error = %v", err)
	}
	if cfg.Interval != 60 {
		t.Errorf("Interval = %d, want 60", cfg.Interval)
	}
}

func TestParseConfig_zeroNode(t *testing.T) {
	var node yaml.Node
	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(zero node) error = %v", err)
	}
	if cfg.Interval != 60 {
		t.Errorf("Interval = %d, want 60", cfg.Interval)
	}
}

func TestParseConfig_partial(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(`interval: 120`), &node); err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(partial) error = %v", err)
	}
	if cfg.Interval != 120 {
		t.Errorf("Interval = %d, want 120", cfg.Interval)
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
