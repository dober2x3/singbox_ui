package prober

import (
	"fmt"
	"log"

	"singbox-config-service/internal/pkg/types"
)

type Service struct {
	prober       *Prober
	config       ProberConfig
	baseDir      string
	resultSaver  ProbeResultSaver
	nodeProvider NodeProvider
}

func NewService(baseDir string, resultSaver ProbeResultSaver) *Service {
	return &Service{
		baseDir:     baseDir,
		resultSaver: resultSaver,
		config:      DefaultProberConfig(),
	}
}

func (s *Service) WithNodeProvider(np NodeProvider) *Service {
	s.nodeProvider = np
	return s
}

func (s *Service) Init() error {
	s.prober = NewProber(s.config)
	if err := s.prober.LoadNodesFromFile(s.baseDir); err != nil {
		log.Printf("Warning: failed to load prober nodes: %v", err)
	}
	return nil
}

func (s *Service) Start() {
	s.prober.Start()
}

func (s *Service) Stop() {
	s.prober.Stop()
}

func (s *Service) AddNode(node types.ProbeNode) {
	s.prober.AddNode(node)
}

func (s *Service) RemoveNode(tag string) {
	s.prober.RemoveNode(tag)
}

func (s *Service) ClearNodes() {
	s.prober.ClearNodes()
}

func (s *Service) UpdateNodes(nodes []types.ProbeNode) {
	s.prober.UpdateNodes(nodes)
}

func (s *Service) SaveNodes() error {
	return s.prober.SaveNodesToFile(s.baseDir)
}

func (s *Service) LoadNodes() error {
	return s.prober.LoadNodesFromFile(s.baseDir)
}

func (s *Service) GetResult(tag string) *types.ProbeResult {
	return s.prober.GetResult(tag)
}

func (s *Service) GetAllResults() map[string]*types.ProbeResult {
	return s.prober.GetAllResults()
}

func (s *Service) GetBestNode() *types.ProbeResult {
	return s.prober.GetBestNode()
}

func (s *Service) GetOnlineNodes() []*types.ProbeResult {
	return s.prober.GetOnlineNodes()
}

func (s *Service) GetStats() map[string]interface{} {
	return s.prober.GetStats()
}

func (s *Service) IsRunning() bool {
	return s.prober.IsRunning()
}

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
