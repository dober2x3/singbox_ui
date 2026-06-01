package prober

import "singbox-config-service/internal/pkg/types"

type ProbeResultSaver interface {
	SaveProbeResults(results []types.ProbeResultUpdate) error
}

type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}
