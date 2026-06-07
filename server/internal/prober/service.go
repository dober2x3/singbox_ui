// Package prober provides node probing, health checking and latency measurement.
package prober

import (
	"fmt"
	"log"

	"singbox-config-service/internal/pkg/types"
)

// Service provides the business-logic layer above the Prober engine.
type Service struct {
	prober       *Prober
	config       Config
	baseDir      string
	resultSaver  ProbeResultSaver
	nodeProvider NodeProvider
}

// NewService creates a new Service with the given config, base directory and result saver.
func NewService(config Config, baseDir string, resultSaver ProbeResultSaver) *Service {
	return &Service{
		baseDir:     baseDir,
		resultSaver: resultSaver,
		config:      config,
	}
}

// WithNodeProvider attaches a NodeProvider for subscription syncing.
func (s *Service) WithNodeProvider(np NodeProvider) *Service {
	s.nodeProvider = np
	return s
}

// Init initialises the underlying prober and loads persisted nodes.
func (s *Service) Init() error {
	s.prober = NewProber(s.config)
	if err := s.prober.LoadNodesFromFile(s.baseDir); err != nil {
		log.Printf("Warning: failed to load prober nodes: %v", err)
	}
	return nil
}

// Start starts the probe loop.
func (s *Service) Start() {
	s.prober.Start()
}

// Stop stops the probe loop.
func (s *Service) Stop() {
	s.prober.Stop()
}

// AddNode registers a new node for probing.
func (s *Service) AddNode(node types.ProbeNode) {
	s.prober.AddNode(node)
}

// RemoveNode unregisters a node by tag.
func (s *Service) RemoveNode(tag string) {
	s.prober.RemoveNode(tag)
}

// ClearNodes removes all registered nodes.
func (s *Service) ClearNodes() {
	s.prober.ClearNodes()
}

// UpdateNodes replaces all registered nodes with the given list.
func (s *Service) UpdateNodes(nodes []types.ProbeNode) {
	s.prober.UpdateNodes(nodes)
}

// SaveNodes persists the current node list to disk.
func (s *Service) SaveNodes() error {
	return s.prober.SaveNodesToFile(s.baseDir)
}

// LoadNodes reads nodes from disk into the prober.
func (s *Service) LoadNodes() error {
	return s.prober.LoadNodesFromFile(s.baseDir)
}

// GetResult returns the latest probe result for the given tag.
func (s *Service) GetResult(tag string) *types.ProbeResult {
	return s.prober.GetResult(tag)
}

// GetAllResults returns all probe results.
func (s *Service) GetAllResults() map[string]*types.ProbeResult {
	return s.prober.GetAllResults()
}

// GetBestNode returns the online node with the lowest latency.
func (s *Service) GetBestNode() *types.ProbeResult {
	return s.prober.GetBestNode()
}

// GetOnlineNodes returns all online nodes.
func (s *Service) GetOnlineNodes() []*types.ProbeResult {
	return s.prober.GetOnlineNodes()
}

// GetStats returns prober statistics and configuration.
func (s *Service) GetStats() map[string]interface{} {
	return s.prober.GetStats()
}

// IsRunning reports whether the probe loop is active.
func (s *Service) IsRunning() bool {
	return s.prober.IsRunning()
}

// SyncNodesFromSubscription fetches nodes from the subscription provider and starts probing.
func (s *Service) SyncNodesFromSubscription() ([]types.ProbeNode, error) {
	if s.nodeProvider == nil {
		return nil, fmt.Errorf("node provider not configured")
	}

	allNodes := s.nodeProvider.GetAllNodes()
	if len(allNodes) == 0 {
		return nil, fmt.Errorf("no nodes in subscription")
	}

	nodes := make([]types.ProbeNode, 0, len(allNodes))
	for _, n := range allNodes {
		tag := ""
		if outbound := n.Outbound; outbound != nil {
			if t, ok := outbound["tag"].(string); ok {
				tag = t
			}
		}
		if tag == "" {
			tag = types.SanitizeTag(n.Protocol, n.Address, n.Port)
		}

		nodes = append(nodes, types.ProbeNode{
			Tag:      tag,
			Protocol: n.Protocol,
			Address:  n.Address,
			Port:     n.Port,
		})
	}

	s.prober.UpdateNodes(nodes)
	if !s.prober.IsRunning() {
		s.prober.Start()
	}

	return nodes, nil
}

// SaveProbeResults persists all current probe results via the result saver.
func (s *Service) SaveProbeResults() (int, error) {
	results := s.prober.GetAllResults()
	if len(results) == 0 {
		return 0, nil
	}

	updates := make([]types.ProbeResultUpdate, 0, len(results))
	for _, r := range results {
		updates = append(updates, types.ProbeResultUpdate{
			Tag:         r.NodeTag,
			Latency:     r.Latency,
			Online:      r.Status == "online",
			LastProbe:   r.LastProbe,
			SuccessRate: int(r.SuccessRate),
		})
	}

	if err := s.resultSaver.SaveProbeResults(updates); err != nil {
		return 0, err
	}

	return len(updates), nil
}
