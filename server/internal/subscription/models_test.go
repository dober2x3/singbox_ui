package subscription

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseConfig_nilNode(t *testing.T) {
	cfg, err := ParseConfig(nil)
	if err != nil {
		t.Fatalf("ParseConfig(nil) error = %v", err)
	}
	if cfg.InsecureTLS {
		t.Error("InsecureTLS should be false by default")
	}
}

func TestParseConfig_zeroNode(t *testing.T) {
	var node yaml.Node
	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig(zero node) error = %v", err)
	}
	if cfg.InsecureTLS {
		t.Error("InsecureTLS should be false by default")
	}
}

func TestParseConfig_insecureTrue(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(`insecure_tls: true`), &node); err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig(&node)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if !cfg.InsecureTLS {
		t.Error("InsecureTLS should be true")
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
