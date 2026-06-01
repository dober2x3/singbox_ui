package docker

import (
	"testing"
)

// TestNewClient verifies that a Docker client can be created (requires Docker daemon).
func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	client.Close()
}

// TestContainerInfo verifies ContainerInfo struct field access.
func TestContainerInfo(t *testing.T) {
	info := ContainerInfo{
		Name:        "test",
		ContainerID: "abc123",
		State:       "running",
		Status:      "Up 1 hour",
		Created:     1000,
	}
	if info.Name != "test" {
		t.Errorf("ContainerInfo.Name = %q, want %q", info.Name, "test")
	}
}

// TestContainerAPI_interface is a compile-time check that *Client implements ContainerAPI.
func TestContainerAPI_interface(t *testing.T) {
	// Compile-time check: *Client implements ContainerAPI
	var _ ContainerAPI = (*Client)(nil)
}
