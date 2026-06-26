package prober

import (
	"encoding/json"
	"os"
	"path/filepath"

	"singbox-config-service/internal/pkg/types"
)

// SaveNodesToFile serialises all registered nodes to a JSON file.
func (p *Prober) SaveNodesToFile(baseDir string) error {
	var nodes []types.ProbeNode
	p.nodes.Range(func(_, value interface{}) bool {
		nodes = append(nodes, value.(types.ProbeNode))
		return true
	})

	data, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(baseDir, "prober_nodes.json")
	return os.WriteFile(filePath, data, 0644)
}

// LoadNodesFromFile reads nodes from a JSON file and replaces the current set.
func (p *Prober) LoadNodesFromFile(baseDir string) error {
	filePath := filepath.Join(baseDir, "prober_nodes.json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var nodes []types.ProbeNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return err
	}

	p.UpdateNodes(nodes)
	return nil
}
