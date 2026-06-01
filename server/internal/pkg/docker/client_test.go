package docker

import (
	"testing"
)

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

func TestContainerAPI_interface(t *testing.T) {
	// Compile-time check: *Client implements ContainerAPI
	var _ ContainerAPI = (*Client)(nil)
}
