package docker

import (
	"context"
	"io"
	"strings"
)

// MockContainerAPI implements ContainerAPI for testing with configurable function fields.
type MockContainerAPI struct {
	CreateContainerFn func(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error)
	StartContainerFn  func(ctx context.Context, containerID string) error
	StopContainerFn   func(ctx context.Context, containerID string, timeout *int) error
	RemoveContainerFn func(ctx context.Context, containerID string, force bool) error
	ContainerLogsFn   func(ctx context.Context, containerID string, tail string) (string, error)
	GetContainerStateFn func(ctx context.Context, containerName string) (string, error)
	ImagePullFn       func(ctx context.Context, image string) (io.ReadCloser, error)
	ImageListFn       func(ctx context.Context, image string) (bool, error)
	CloseFn           func() error
}

// NewMockClient creates a MockContainerAPI with default successful responses.
func NewMockClient() *MockContainerAPI {
	return &MockContainerAPI{
		CreateContainerFn: func(_ context.Context, config interface{}, hostConfig interface{}, name string) (string, error) {
			return "mock-container-id", nil
		},
		StartContainerFn: func(_ context.Context, containerID string) error {
			return nil
		},
		StopContainerFn: func(_ context.Context, containerID string, timeout *int) error {
			return nil
		},
		RemoveContainerFn: func(_ context.Context, containerID string, force bool) error {
			return nil
		},
		ContainerLogsFn: func(_ context.Context, containerID string, tail string) (string, error) {
			return "", nil
		},
		GetContainerStateFn: func(_ context.Context, containerName string) (string, error) {
			return "running", nil
		},
		ImagePullFn: func(_ context.Context, image string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("")), nil
		},
		ImageListFn: func(_ context.Context, image string) (bool, error) {
			return true, nil
		},
		CloseFn: func() error {
			return nil
		},
	}
}

// ContainerCreate delegates to the mock function.
func (m *MockContainerAPI) ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error) {
	return m.CreateContainerFn(ctx, config, hostConfig, name)
}

// ContainerStart delegates to the mock function.
func (m *MockContainerAPI) ContainerStart(ctx context.Context, containerID string) error {
	return m.StartContainerFn(ctx, containerID)
}

// ContainerStop delegates to the mock function.
func (m *MockContainerAPI) ContainerStop(ctx context.Context, containerID string, timeout *int) error {
	return m.StopContainerFn(ctx, containerID, timeout)
}

// ContainerRemove delegates to the mock function.
func (m *MockContainerAPI) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	return m.RemoveContainerFn(ctx, containerID, force)
}

// ContainerLogs delegates to the mock function.
func (m *MockContainerAPI) ContainerLogs(ctx context.Context, containerID string, tail string) (string, error) {
	return m.ContainerLogsFn(ctx, containerID, tail)
}

// GetContainerState delegates to the mock function.
func (m *MockContainerAPI) GetContainerState(ctx context.Context, containerName string) (string, error) {
	return m.GetContainerStateFn(ctx, containerName)
}

// ImagePull delegates to the mock function.
func (m *MockContainerAPI) ImagePull(ctx context.Context, image string) (io.ReadCloser, error) {
	return m.ImagePullFn(ctx, image)
}

// ImageList delegates to the mock function.
func (m *MockContainerAPI) ImageList(ctx context.Context, image string) (bool, error) {
	return m.ImageListFn(ctx, image)
}

// Close delegates to the mock function.
func (m *MockContainerAPI) Close() error {
	return m.CloseFn()
}
