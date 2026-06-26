package prober

import (
	"log"

	"singbox-config-service/internal/pkg/types"
)

// AddNode registers a new node for probing and initialises its result state.
func (p *Prober) AddNode(node types.ProbeNode) {
	p.nodes.Store(node.Tag, node)

	result := &types.ProbeResult{
		NodeTag:   node.Tag,
		Protocol:  node.Protocol,
		Address:   node.Address,
		Port:      node.Port,
		Latency:   -1,
		Status:    "unknown",
		LastProbe: "",
	}
	p.results.Store(node.Tag, result)

	history := &nodeHistory{
		results: make([]bool, p.config.MaxResults),
		index:   0,
		size:    p.config.MaxResults,
	}
	p.history.Store(node.Tag, history)

	log.Printf("Prober: added node %s (%s://%s:%d)", node.Tag, node.Protocol, node.Address, node.Port)
}

// RemoveNode unregisters a node by tag and deletes its results.
func (p *Prober) RemoveNode(tag string) {
	p.nodes.Delete(tag)
	p.results.Delete(tag)
	p.history.Delete(tag)
	log.Printf("Prober: removed node %s", tag)
}

// ClearNodes removes all registered nodes and their results.
func (p *Prober) ClearNodes() {
	p.nodes.Range(func(key, _ interface{}) bool {
		p.nodes.Delete(key)
		return true
	})
	p.results.Range(func(key, _ interface{}) bool {
		p.results.Delete(key)
		return true
	})
	p.history.Range(func(key, _ interface{}) bool {
		p.history.Delete(key)
		return true
	})
	log.Println("Prober: cleared all nodes")
}

// UpdateNodes replaces all registered nodes with the given list.
func (p *Prober) UpdateNodes(nodes []types.ProbeNode) {
	p.ClearNodes()
	for _, node := range nodes {
		p.AddNode(node)
	}
	log.Printf("Prober: updated with %d nodes", len(nodes))
}
