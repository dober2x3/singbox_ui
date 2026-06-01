package singbox

import (
	"context"
	"io"

	"singbox-config-service/internal/pkg/docker"
)

type ContainerManager interface {
	ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (containerID string, err error)
	ContainerStart(ctx context.Context, containerID string) error
	ContainerStop(ctx context.Context, containerID string, timeout *int) error
	ContainerRemove(ctx context.Context, containerID string, force bool) error
	ContainerLogs(ctx context.Context, containerID string, tail string) (string, error)
	GetContainerState(ctx context.Context, containerName string) (state string, err error)
	ImagePull(ctx context.Context, image string) (io.ReadCloser, error)
	ImageList(ctx context.Context, image string) (bool, error)
	ListContainers(ctx context.Context, prefix string) ([]docker.ContainerInfo, error)
	EnsureImage(imageName, tarPath string) error
	Close() error
}
