package speedtest

import (
	"context"

	"singbox-config-service/internal/pkg/types"
)

type ContainerManager interface {
	ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (containerID string, err error)
	ContainerStart(ctx context.Context, containerID string) error
	ContainerStop(ctx context.Context, containerID string, timeout *int) error
	ContainerRemove(ctx context.Context, containerID string, force bool) error
}

type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}

type SpeedTestResultSaver interface {
	SaveSpeedTestResults(results []types.SpeedTestUpdate) error
}
