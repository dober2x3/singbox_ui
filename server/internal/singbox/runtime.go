// Package singbox provides services for managing sing-box configuration and containers.
package singbox

import "context"

// Runtime abstracts the lifecycle of a sing-box instance.
// Implementations: DockerRuntime (via Docker SDK), NativeRuntime (via os/exec).
type Runtime interface {
	// Start launches an instance with the given name and config file path.
	// Returns an opaque instance identifier (container ID or "pid:<N>").
	Start(ctx context.Context, name string, configPath string) (id string, err error)

	// Stop terminates an instance gracefully within the optional timeout (seconds).
	// If timeout is nil, a default timeout applies.
	Stop(ctx context.Context, name string, timeout *int) error

	// Status reports whether an instance is running and its identifier.
	Status(ctx context.Context, name string) (running bool, id string, err error)

	// Logs returns recent log lines from an instance.
	// tail specifies the number of lines (empty defaults to 100).
	Logs(ctx context.Context, name string, tail string) (string, error)

	// Version returns the sing-box version string.
	Version(ctx context.Context) (string, error)

	// List returns all instances managed by this runtime.
	List(ctx context.Context) ([]InstanceInfo, error)

	// Close releases any underlying resources (e.g. Docker client).
	Close() error
}

// InstanceInfo describes a sing-box instance.
type InstanceInfo struct {
	Name    string `json:"name"`
	ID      string `json:"containerId,omitempty"`
	Running bool   `json:"running"`
	State   string `json:"state,omitempty"`
}
