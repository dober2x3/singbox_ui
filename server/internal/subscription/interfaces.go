package subscription

import "singbox-config-service/internal/pkg/types"

// SubscriptionUpdater provides methods to load and update individual subscriptions (e.g. for scheduler use).
type SubscriptionUpdater interface {
	LoadAll() ([]SubscriptionEntry, error)
	UpdateOne(id string) (*SubscriptionEntry, error)
}

// NodeProvider provides access to all proxy nodes.
type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}

// ProbeResultSaver persists probe (latency) results to the underlying store.
type ProbeResultSaver interface {
	SaveProbeResults(results []types.ProbeResultUpdate) error
}

// SpeedTestResultSaver persists speed test results to the underlying store.
type SpeedTestResultSaver interface {
	SaveSpeedTestResults(results []types.SpeedTestUpdate) error
}
