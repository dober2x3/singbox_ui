//go:build docker

package tunnelrunner

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
)

const speedTestContainerName = "sing-box-speedtest"
const singBoxImage = "ghcr.io/sagernet/sing-box:v1.13.5"

type dockerRunner struct {
	client *docker.Client
	cfg    *config.AppConfig
}

func NewRunner(cfg *config.AppConfig) Runner {
	client, err := docker.NewClient()
	if err != nil {
		return &dockerRunner{cfg: cfg}
	}
	return &dockerRunner{client: client, cfg: cfg}
}

func (d *dockerRunner) StartTemp(ctx context.Context, configPath string) (string, error) {
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

func (d *dockerRunner) StopTemp(ctx context.Context, id string) error {
	d.cleanupContainer(ctx)
	return nil
}

func (d *dockerRunner) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	return waitProxyReady(ctx, port, timeout)
}

func (d *dockerRunner) GetTempLogs(ctx context.Context, id string) string {
	if d.client == nil {
		return "Docker client not available"
	}
	logs, err := d.client.ContainerLogs(ctx, speedTestContainerName, "50")
	if err != nil {
		return fmt.Sprintf("Error getting logs: %v", err)
	}
	return logs
}

func (d *dockerRunner) cleanupContainer(ctx context.Context) {
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

func waitProxyReady(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}
