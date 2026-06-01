package subscription

import "singbox-config-service/internal/pkg/types"

type SubscriptionUpdater interface {
	LoadAll() ([]SubscriptionEntry, error)
	UpdateOne(id string) (*SubscriptionEntry, error)
}

type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}

type ProbeResultSaver interface {
	SaveProbeResults(results []types.ProbeResultUpdate) error
}

type SpeedTestResultSaver interface {
	SaveSpeedTestResults(results []types.SpeedTestUpdate) error
}
