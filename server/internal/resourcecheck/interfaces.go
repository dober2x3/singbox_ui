package resourcecheck

import (
	"context"
	"time"

	"singbox-config-service/internal/pkg/types"
)

// Runner is the tunnel lifecycle manager.
type Runner interface {
	StartTemp(ctx context.Context, configPath string) (string, error)
	StopTemp(ctx context.Context, id string) error
	WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error
	GetTempLogs(ctx context.Context, id string) string
}

// NodeProvider provides proxy nodes to check resources through.
type NodeProvider interface {
	GetAllNodes() []types.ProxyNode
}
