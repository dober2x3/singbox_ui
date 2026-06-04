package speedtest

import "singbox-config-service/internal/pkg/types"

// NodeProvider provides proxy nodes to test.
type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}

// SpeedTestResultSaver persists speed test results.
type SpeedTestResultSaver interface {
	SaveSpeedTestResults(results []types.SpeedTestUpdate) error
}
