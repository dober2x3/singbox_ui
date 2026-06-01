package docker

import (
	"context"
	"io"
)

// ContainerAPI defines the interface for Docker container and image operations.
type ContainerAPI interface {
	ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (containerID string, err error)
	ContainerStart(ctx context.Context, containerID string) error
	ContainerStop(ctx context.Context, containerID string, timeout *int) error
	ContainerRemove(ctx context.Context, containerID string, force bool) error
	ContainerLogs(ctx context.Context, containerID string, tail string) (string, error)
	GetContainerState(ctx context.Context, containerName string) (state string, err error)
	ImagePull(ctx context.Context, image string) (io.ReadCloser, error)
	ImageList(ctx context.Context, image string) (bool, error)
	Close() error
}
