// Package tunnelrunner manages temporary sing-box proxy instances (tunnels)
// used for speed tests and resource availability checks.
package tunnelrunner

import (
	"context"
	"time"
)

// Runner manages the lifecycle of a temporary proxy (tunnel) instance.
type Runner interface {
	// StartTemp launches a temporary proxy from the given config and returns an instance ID.
	StartTemp(ctx context.Context, configPath string) (string, error)
	// StopTemp stops a running temporary proxy instance.
	StopTemp(ctx context.Context, id string) error
	// WaitTempReady blocks until the proxy is accepting connections on the given port.
	WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error
	// GetTempLogs returns the logs of a temporary proxy instance.
	GetTempLogs(ctx context.Context, id string) string
}
