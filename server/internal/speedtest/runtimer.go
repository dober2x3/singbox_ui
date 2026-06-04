// Package speedtest provides a speed testing service for proxy nodes.
package speedtest

import (
	"context"
	"time"
)

// TempRuntime manages short-lived sing-box instances for speed testing.
type TempRuntime interface {
	// StartTemp starts a temporary sing-box instance with the given config path.
	StartTemp(ctx context.Context, configPath string) (id string, err error)

	// StopTemp stops and cleans up a temporary instance.
	StopTemp(ctx context.Context, id string) error

	// WaitTempReady blocks until the instance accepts TCP connections on the given port.
	WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error

	// GetTempLogs returns collected log output from a temporary instance.
	GetTempLogs(ctx context.Context, id string) string
}
