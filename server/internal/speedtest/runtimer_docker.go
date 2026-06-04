//go:build !openwrt

package speedtest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
)

const speedTestContainerName = "sing-box-speedtest"
const singBoxImage = "ghcr.io/sagernet/sing-box:v1.13.5"

// DockerTempRuntime creates temporary sing-box containers for speed tests.
type DockerTempRuntime struct {
	client *docker.Client
	cfg    *config.Config
}

// NewTempRuntime creates a TempRuntime backed by Docker containers.
func NewTempRuntime(cfg *config.Config) TempRuntime {
	client, err := docker.NewClient()
	if err != nil {
		return &DockerTempRuntime{cfg: cfg}
	}
	return &DockerTempRuntime{client: client, cfg: cfg}
}

func (d *DockerTempRuntime) StartTemp(ctx context.Context, configPath string) (string, error) {
	if d.client == nil {
		return "", fmt.Errorf("Docker client not available")
	}
	hostConfigPath := configPath
	if resolved, err := d.cfg.ResolveHostConfigDir(configPath); err == nil {
		hostConfigPath = resolved
	}

	containerConfig := map[string]interface{}{
		"Image": singBoxImage,
		"Cmd":   []string{"-D", "/var/lib/sing-box", "-C", "/etc/sing-box/", "run"},
	}
	hostConfig := map[string]interface{}{
		"Binds":       []string{hostConfigPath + ":/etc/sing-box/config.json:ro"},
		"NetworkMode": "host",
		"CapAdd":      []string{"NET_ADMIN"},
	}

	d.cleanupContainer(ctx)

	for attempt := 0; attempt < 5; attempt++ {
		id, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, speedTestContainerName)
		if err == nil {
			if err := d.client.ContainerStart(ctx, id); err != nil {
				d.cleanupContainer(ctx)
				continue
			}
			return id, nil
		}
		if !isConflictError(err) {
			d.cleanupContainer(ctx)
			continue
		}
		time.Sleep(200 * time.Millisecond)
	}
	return "", fmt.Errorf("failed to create speedtest container after retries")
}

func (d *DockerTempRuntime) StopTemp(ctx context.Context, id string) error {
	d.cleanupContainer(ctx)
	return nil
}

func (d *DockerTempRuntime) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	return waitProxyReady(ctx, port, timeout)
}

func (d *DockerTempRuntime) GetTempLogs(ctx context.Context, id string) string {
	if d.client == nil {
		return "Docker client not available"
	}
	logs, err := d.client.ContainerLogs(ctx, speedTestContainerName, "50")
	if err != nil {
		return fmt.Sprintf("Error getting logs: %v", err)
	}
	return logs
}

func (d *DockerTempRuntime) cleanupContainer(ctx context.Context) {
	if d.client == nil {
		return
	}
	_ = d.client.ContainerRemove(ctx, speedTestContainerName, true)
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "already in use") || strings.Contains(s, "Conflict")
}
