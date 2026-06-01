package prober

import "singbox-config-service/internal/pkg/types"

// ProbeResultSaver defines the interface for persisting probe results.
type ProbeResultSaver interface {
	SaveProbeResults(results []types.ProbeResultUpdate) error
}

// NodeProvider defines the interface for retrieving proxy nodes.
type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}
