package docker

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Client{
		cli: cli,
	}, nil
}

func (d *Client) Close() error {
	if d.cli != nil {
		return d.cli.Close()
	}
	return nil
}

// EnsureImage ensures the image exists, trying load from tar first then pull
func (d *Client) EnsureImage(ctx context.Context, imageName, tarPath string) error {
	log.Printf("Checking if image %s exists...", imageName)

	images, err := d.cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageName)),
	})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}
	if len(images) > 0 {
		log.Printf("Image %s already exists", imageName)
		return nil
	}

	// Try loading from tar
	if tarPath != "" {
		if _, err := os.Stat(tarPath); err == nil {
			if err := d.loadImageFromFile(ctx, tarPath, imageName); err == nil {
				return nil
			} else {
				log.Printf("Embedded image load failed: %v, falling back to pull", err)
			}
		}
	}

	// Pull from registry
	log.Printf("Pulling image %s...", imageName)
	reader, err := d.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}
	log.Printf("Image %s pulled successfully", imageName)
	return nil
}

func (d *Client) loadImageFromFile(ctx context.Context, tarPath, imageName string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := d.cli.ImageLoad(ctx, file, true)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	// Re-tag CI temp tag if needed
	images, err := d.cli.ImageList(ctx, types.ImageListOptions{})
	if err == nil {
		for _, img := range images {
			for _, tag := range img.RepoTags {
				if strings.HasPrefix(tag, "singbox:") {
					if err := d.cli.ImageTag(ctx, img.ID, imageName); err != nil {
						log.Printf("Warning: failed to re-tag image: %v", err)
					}
					break
				}
			}
		}
	}

	// Verify
	verifyImages, err := d.cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageName)),
	})
	if err != nil || len(verifyImages) == 0 {
		return fmt.Errorf("image loaded but not found under expected tag %s", imageName)
	}

	os.Remove(tarPath)
	log.Printf("Image loaded from %s successfully", tarPath)
	return nil
}

func (d *Client) ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error) {
	cfg, ok := config.(*container.Config)
	if !ok {
		return "", fmt.Errorf("config must be *container.Config")
	}
	hcfg, ok := hostConfig.(*container.HostConfig)
	if !ok {
		return "", fmt.Errorf("hostConfig must be *container.HostConfig")
	}
	resp, err := d.cli.ContainerCreate(ctx, cfg, hcfg, nil, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	return resp.ID, nil
}

func (d *Client) ContainerStart(ctx context.Context, containerID string) error {
	if err := d.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

func (d *Client) ContainerStop(ctx context.Context, containerID string, timeout *int) error {
	stopOptions := container.StopOptions{Timeout: timeout}
	if err := d.cli.ContainerStop(ctx, containerID, stopOptions); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}
	return nil
}

func (d *Client) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	if err := d.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: force}); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}
	return nil
}

func (d *Client) ContainerLogs(ctx context.Context, containerID, tail string) (string, error) {
	if tail == "" {
		tail = "100"
	}
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	}
	reader, err := d.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, reader)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	logs := stdout.String()
	if stderr.Len() > 0 {
		logs += "\n--- STDERR ---\n" + stderr.String()
	}
	return logs, nil
}

func (d *Client) GetContainerState(ctx context.Context, containerName string) (state string, err error) {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", containerName)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	if len(containers) == 0 {
		return "", nil
	}
	return containers[0].State, nil
}

func (d *Client) ListContainers(ctx context.Context, prefix string) ([]ContainerInfo, error) {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", prefix)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		result = append(result, ContainerInfo{
			Name:        name,
			ContainerID: c.ID[:12],
			State:       c.State,
			Status:      c.Status,
			Created:     c.Created,
		})
	}
	return result, nil
}

// ImageList checks if an image exists locally
func (d *Client) ImageList(ctx context.Context, image string) (bool, error) {
	images, err := d.cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", image)),
	})
	if err != nil {
		return false, err
	}
	return len(images) > 0, nil
}

// ImagePull pulls an image
func (d *Client) ImagePull(ctx context.Context, image string) (io.ReadCloser, error) {
	return d.cli.ImagePull(ctx, image, types.ImagePullOptions{})
}
