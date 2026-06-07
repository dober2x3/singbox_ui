// Package singbox provides services for managing sing-box configuration and containers.
//
//go:build docker

package singbox

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types/container"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
)

// DockerRuntime manages sing-box instances via Docker containers.
type DockerRuntime struct {
	client *docker.Client
	cfg    *config.AppConfig
}

// NewRuntime creates a Runtime backed by Docker.
func NewRuntime(cfg *config.AppConfig) (Runtime, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	rt := &DockerRuntime{client: client, cfg: cfg}
	go func() {
		log.Println("Ensuring sing-box Docker image...")
		if err := client.EnsureImage(context.Background(), "ghcr.io/sagernet/sing-box:latest", ""); err != nil {
			log.Printf("Warning: failed to ensure sing-box image: %v", err)
		}
	}()
	return rt, nil
}

func (d *DockerRuntime) Start(ctx context.Context, name, configPath string) (string, error) {
	containerName := "singbox-" + name

	state, _ := d.client.GetContainerState(ctx, containerName)
	if state == "running" {
		return state, fmt.Errorf("container %s is already running", containerName)
	}
	if state != "" {
		_ = d.client.ContainerRemove(ctx, containerName, true)
	}

	hostConfigPath := configPath
	if resolved, err := d.cfg.ResolveHostConfigDir(configPath); err == nil {
		hostConfigPath = resolved
	}

	containerConfig := &container.Config{
		Image: "ghcr.io/sagernet/sing-box:latest",
		Cmd:   []string{"run", "-c", "/etc/sing-box/config.json"},
	}
	hostConfig := &container.HostConfig{
		Binds:       []string{hostConfigPath + ":/etc/sing-box/config.json:ro"},
		NetworkMode: container.NetworkMode("host"),
		CapAdd:      []string{"NET_ADMIN", "SYS_MODULE"},
	}

	id, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	if err := d.client.ContainerStart(ctx, id); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	return id, nil
}

func (d *DockerRuntime) Stop(ctx context.Context, name string, timeout *int) error {
	containerName := "singbox-" + name
	state, err := d.client.GetContainerState(ctx, containerName)
	if err != nil {
		return err
	}
	if state == "" {
		return nil
	}
	t := 10
	if timeout != nil {
		t = *timeout
	}
	return d.client.ContainerStop(ctx, containerName, &t)
}

func (d *DockerRuntime) Status(ctx context.Context, name string) (bool, string, error) {
	containerName := "singbox-" + name
	containers, err := d.client.ListContainers(ctx, containerName)
	if err != nil {
		return false, "", err
	}
	if len(containers) == 0 {
		return false, "", nil
	}
	return containers[0].State == "running", containers[0].ContainerID, nil
}

func (d *DockerRuntime) Logs(ctx context.Context, name, tail string) (string, error) {
	containerName := "singbox-" + name
	if tail == "" {
		tail = "100"
	}
	return d.client.ContainerLogs(ctx, containerName, tail)
}

func (d *DockerRuntime) Version(_ context.Context) (string, error) {
	return "sing-box 1.10.0", nil
}

func (d *DockerRuntime) List(ctx context.Context) ([]InstanceInfo, error) {
	containers, err := d.client.ListContainers(ctx, "singbox")
	if err != nil {
		return nil, err
	}
	result := make([]InstanceInfo, len(containers))
	for i, c := range containers {
		result[i] = InstanceInfo{
			Name:    c.Name,
			ID:      c.ContainerID,
			Running: c.State == "running",
			State:   c.State,
		}
	}
	return result, nil
}

func (d *DockerRuntime) Close() error {
	return d.client.Close()
}
