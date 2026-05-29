package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Docker container config constants
const (
	SingBoxContainerName   = "sing-box"
	SingBoxContainerPrefix = "sing-box-" // used for multi-config container naming
	SingBoxVersion         = "v1.13.5"
	SingBoxImageName       = "ghcr.io/sagernet/sing-box:" + SingBoxVersion
	SingBoxImageTarPath    = "/root/singbox-image.tar" // CI pre-pulled image tar path
	ContainerConfigDir     = "/etc/sing-box"
	ContainerDataDir       = "/var/lib/sing-box"
)


// DockerService Docker service wrapper
type DockerService struct {
	cli *client.Client
	ctx context.Context
}

// NewDockerService creates a Docker service instance
func NewDockerService() (*DockerService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &DockerService{
		cli: cli,
		ctx: context.Background(),
	}, nil
}

// Close closes the Docker client
func (d *DockerService) Close() error {
	if d.cli != nil {
		return d.cli.Close()
	}
	return nil
}

// getHostDataDir gets the host data directory path (via HOST_DATA_DIR env var)
func getHostDataDir() string {
	return os.Getenv("HOST_DATA_DIR")
}

// resolveHostConfigDir resolves container-internal path to host path
// Used for volume mounts when creating sing-box container in Docker-in-Docker scenarios
func resolveHostConfigDir(containerPath string) (string, error) {
	hostDir := getHostDataDir()
	if hostDir == "" {
		return "", fmt.Errorf("HOST_DATA_DIR environment variable is not set")
	}

	containerDataDir := os.Getenv("DATA_DIR")
	if containerDataDir == "" {
		containerDataDir = "/home/data"
	}

	// Replace container path /home/data/xxx with host path/xxx
	rel, err := filepath.Rel(containerDataDir, containerPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is not under DATA_DIR %s", containerPath, containerDataDir)
	}
	return filepath.Join(hostDir, rel), nil
}

// EnsureImage ensures the image exists: prefer loading from embedded tar, otherwise pull from remote
func (d *DockerService) EnsureImage() error {
	log.Printf("Checking if image %s exists...", SingBoxImageName)

	// Check if image already exists
	images, err := d.cli.ImageList(d.ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", SingBoxImageName)),
	})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	if len(images) > 0 {
		log.Printf("Image %s already exists", SingBoxImageName)
		return nil
	}

	// Try to load from embedded tar file
	if err := d.loadImageFromFile(); err == nil {
		return nil
	} else {
		log.Printf("No embedded image or load failed: %v, falling back to pull", err)
	}

	// Pull from remote
	log.Printf("Pulling image %s...", SingBoxImageName)
	reader, err := d.cli.ImagePull(d.ctx, SingBoxImageName, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read pull response: %w", err)
	}

	log.Printf("Image %s pulled successfully", SingBoxImageName)
	return nil
}

// loadImageFromFile loads Docker image from embedded tar file
func (d *DockerService) loadImageFromFile() error {
	tarPath := SingBoxImageTarPath
	if _, err := os.Stat(tarPath); os.IsNotExist(err) {
		return fmt.Errorf("image tar not found: %s", tarPath)
	}

	log.Printf("Loading image from %s...", tarPath)
	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open image tar: %w", err)
	}
	defer file.Close()

	resp, err := d.cli.ImageLoad(d.ctx, file, true)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// Tag inside tar may differ from expected (CI uses temp tag), need to re-tag
	// Find the just-loaded image and ensure SingBoxImageName tag exists
	images, err := d.cli.ImageList(d.ctx, types.ImageListOptions{})
	if err == nil {
		for _, img := range images {
			for _, tag := range img.RepoTags {
				if strings.HasPrefix(tag, "singbox:") {
					// CI-generated temp tag, re-tag to official name
					if err := d.cli.ImageTag(d.ctx, img.ID, SingBoxImageName); err != nil {
						log.Printf("Warning: failed to re-tag image: %v", err)
					} else {
						log.Printf("Re-tagged %s -> %s", tag, SingBoxImageName)
					}
					break
				}
			}
		}
	}

	// Verify image loaded correctly
	verifyImages, err := d.cli.ImageList(d.ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", SingBoxImageName)),
	})
	if err != nil || len(verifyImages) == 0 {
		return fmt.Errorf("image loaded but not found under expected tag %s", SingBoxImageName)
	}

	// Delete tar file after successful load to free disk space
	if err := os.Remove(tarPath); err != nil {
		log.Printf("Warning: failed to remove image tar %s: %v", tarPath, err)
	}

	log.Printf("Image loaded from %s successfully", tarPath)
	return nil
}

// CreateAndStartContainer creates and starts a sing-box container
func (d *DockerService) CreateAndStartContainer(hostConfigDir string) (string, error) {
	// First try to remove any container with the same name
	_ = d.RemoveContainer()

	// Resolve container-internal path to host path (Docker-in-Docker scenario)
	hostSingboxDir, err := resolveHostConfigDir(hostConfigDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve host config dir: %w", err)
	}
	log.Printf("Mounting host directory %s to container /etc/sing-box", hostSingboxDir)

	// Ensure ACME data directory exists (using container-internal path)
	internalAcmeDir := filepath.Join(hostConfigDir, "acme")
	if err := os.MkdirAll(internalAcmeDir, 0755); err != nil {
		log.Printf("Warning: failed to create ACME directory %s: %v", internalAcmeDir, err)
	}
	hostAcmeDir := hostSingboxDir + "/acme"

	// Container config
	// sing-box image uses sing-box as entrypoint
	// Command format: -D /var/lib/sing-box -C /etc/sing-box/ run
	config := &container.Config{
		Image: SingBoxImageName,
		Cmd:   []string{"-D", ContainerDataDir, "-C", ContainerConfigDir + "/", "run"},
	}

	// Host config
	hostConfig := &container.HostConfig{
		// Use host network mode
		NetworkMode: "host",

		// Config file mounts
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   hostSingboxDir,
				Target:   "/etc/sing-box",
				ReadOnly: true,
			},
			{
				// ACME data directory (for automatic certificate requests)
				Type:     mount.TypeBind,
				Source:   hostAcmeDir,
				Target:   "/var/lib/sing-box/acme",
				ReadOnly: false,
			},
		},

		// Add NET_ADMIN capability (required by sing-box)
		CapAdd: []string{"NET_ADMIN"},

		// Restart policy
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},

		// Resource limits
		Resources: container.Resources{
			Memory:   512 * 1024 * 1024, // 512MB
			NanoCPUs: 1000000000,        // 1 CPU
		},
	}

	// Create container
	resp, err := d.cli.ContainerCreate(
		d.ctx,
		config,
		hostConfig,
		nil, // networkingConfig
		nil, // platform
		SingBoxContainerName,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := d.cli.ContainerStart(d.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		// If start fails, remove container
		_ = d.cli.ContainerRemove(d.ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("Container %s started with ID: %s", SingBoxContainerName, resp.ID[:12])
	return resp.ID, nil
}

// StopContainer stops the sing-box container
func (d *DockerService) StopContainer() error {
	timeout := 10 // 10 second timeout
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := d.cli.ContainerStop(d.ctx, SingBoxContainerName, stopOptions); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	log.Printf("Container %s stopped", SingBoxContainerName)
	return nil
}

// RemoveContainer removes the sing-box container
func (d *DockerService) RemoveContainer() error {
	if err := d.cli.ContainerRemove(d.ctx, SingBoxContainerName, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		// Ignore if container does not exist
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	log.Printf("Container %s removed", SingBoxContainerName)
	return nil
}

// GetContainerStatus gets the container status
func (d *DockerService) GetContainerStatus() (running bool, containerID string, err error) {
	containers, err := d.cli.ContainerList(d.ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", SingBoxContainerName)),
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return false, "", nil
	}

	c := containers[0]
	return c.State == "running", c.ID, nil
}

// GetContainerLogs gets the container logs
func (d *DockerService) GetContainerLogs(tail string) (string, error) {
	if tail == "" {
		tail = "100"
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	}

	reader, err := d.cli.ContainerLogs(d.ctx, SingBoxContainerName, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Use stdcopy to separate stdout and stderr
	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, reader)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	// Merge output
	logs := stdout.String()
	if stderr.Len() > 0 {
		logs += "\n--- STDERR ---\n" + stderr.String()
	}

	return logs, nil
}

// GetSingBoxVersion creates a temp container to run `sing-box version`
func (d *DockerService) GetSingBoxVersion() (string, error) {
	resp, err := d.cli.ContainerCreate(d.ctx, &container.Config{
		Image: SingBoxImageName,
		Cmd:   []string{"version"},
	}, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create temp container: %w", err)
	}
	defer d.cli.ContainerRemove(d.ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	if err := d.cli.ContainerStart(d.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start temp container: %w", err)
	}

	// Wait for container to exit
	statusCh, errCh := d.cli.ContainerWait(d.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error waiting for container: %w", err)
		}
	case <-statusCh:
	case <-time.After(10 * time.Second):
		return "", fmt.Errorf("timeout waiting for version")
	}

	// Read output from container logs (container already exited, read all at once)
	logReader, err := d.cli.ContainerLogs(d.ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}
	defer logReader.Close()

	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, logReader)
	if err != nil {
		return "", fmt.Errorf("failed to parse logs: %w", err)
	}

	// Only take first line: sing-box version x.x.x
	output := strings.TrimSpace(stdout.String())
	if i := strings.IndexByte(output, '\n'); i != -1 {
		output = output[:i]
	}
	return strings.TrimSpace(output), nil
}

// CheckNamedConfig validates the named config using a temporary container
func (d *DockerService) CheckNamedConfig(configName string, hostConfigDir string) (bool, string, error) {
	hostSingboxDir, err := resolveHostConfigDir(hostConfigDir)
	if err != nil {
		return false, "", fmt.Errorf("failed to resolve host config dir: %w", err)
	}

	resp, err := d.cli.ContainerCreate(d.ctx, &container.Config{
		Image: SingBoxImageName,
		Cmd:   []string{"check", "-C", ContainerConfigDir + "/"},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   hostSingboxDir,
				Target:   ContainerConfigDir,
				ReadOnly: true,
			},
		},
	}, nil, nil, "")
	if err != nil {
		return false, "", fmt.Errorf("failed to create check container: %w", err)
	}
	defer d.cli.ContainerRemove(d.ctx, resp.ID, types.ContainerRemoveOptions{Force: true})

	if err := d.cli.ContainerStart(d.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return false, "", fmt.Errorf("failed to start check container: %w", err)
	}

	// Wait for container to exit
	statusCh, errCh := d.cli.ContainerWait(d.ctx, resp.ID, container.WaitConditionNotRunning)
	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			return false, "", fmt.Errorf("error waiting for check container: %w", err)
		}
		// errCh returned nil, still need to read status
		status := <-statusCh
		exitCode = status.StatusCode
	case status := <-statusCh:
		exitCode = status.StatusCode
	case <-time.After(30 * time.Second):
		return false, "", fmt.Errorf("timeout waiting for config check")
	}

	// Read output
	logReader, err := d.cli.ContainerLogs(d.ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to read check logs: %w", err)
	}
	defer logReader.Close()

	var stdout, stderr strings.Builder
	_, _ = stdcopy.StdCopy(&stdout, &stderr, logReader)

	output := strings.TrimSpace(stderr.String())
	if output == "" {
		output = strings.TrimSpace(stdout.String())
	}

	// Remove ANSI escape codes
	output = stripAnsi(output)

	if exitCode != 0 {
		return false, output, nil
	}
	return true, output, nil
}

// SpeedTestContainerName temporary speed test container name
const SpeedTestContainerName = "sing-box-speedtest"

// StartSpeedTestContainer starts a temporary container for proxy speed testing
func (d *DockerService) StartSpeedTestContainer(hostConfigDir string) error {
	_ = d.StopSpeedTestContainer()

	hostSingboxDir, err := resolveHostConfigDir(hostConfigDir)
	if err != nil {
		return fmt.Errorf("resolve host dir: %w", err)
	}

	createCfg := &container.Config{
		Image: SingBoxImageName,
		Cmd:   []string{"-D", ContainerDataDir, "-C", ContainerConfigDir + "/", "run"},
	}
	hostCfg := &container.HostConfig{
		NetworkMode: "host",
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   hostSingboxDir,
				Target:   ContainerConfigDir,
				ReadOnly: true,
			},
		},
		CapAdd: []string{"NET_ADMIN"},
	}

	// After ContainerRemove returns, Docker daemon may not have released the container name yet,
	// retry to avoid "container name already in use" race condition
	var resp container.CreateResponse
	for attempt := 0; attempt < 5; attempt++ {
		resp, err = d.cli.ContainerCreate(d.ctx, createCfg, hostCfg, nil, nil, SpeedTestContainerName)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "already in use") && !strings.Contains(err.Error(), "Conflict") {
			return fmt.Errorf("create: %w", err)
		}
		_ = d.StopSpeedTestContainer()
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("create after retries: %w", err)
	}

	if err := d.cli.ContainerStart(d.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		_ = d.cli.ContainerRemove(d.ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return fmt.Errorf("start: %w", err)
	}
	return nil
}

// GetSpeedTestContainerLogs gets speed test container logs (for diagnosing startup failures)
func (d *DockerService) GetSpeedTestContainerLogs() string {
	reader, err := d.cli.ContainerLogs(d.ctx, SpeedTestContainerName, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "30",
	})
	if err != nil {
		return fmt.Sprintf("(failed to get logs: %v)", err)
	}
	defer reader.Close()
	var stdout, stderr strings.Builder
	_, _ = stdcopy.StdCopy(&stdout, &stderr, reader)
	out := strings.TrimSpace(stderr.String())
	if out == "" {
		out = strings.TrimSpace(stdout.String())
	}
	return stripAnsi(out)
}

// StopSpeedTestContainer stops and removes the speed test container
func (d *DockerService) StopSpeedTestContainer() error {
	if err := d.cli.ContainerRemove(d.ctx, SpeedTestContainerName, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil && !strings.Contains(err.Error(), "No such container") {
		return err
	}
	return nil
}

// stripAnsi removes ANSI escape sequences from a string
func stripAnsi(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// execInContainer executes a command in a running container
func (d *DockerService) execInContainer(cmd ...string) (string, error) {
	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := d.cli.ContainerExecCreate(d.ctx, SingBoxContainerName, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := d.cli.ContainerExecAttach(d.ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()

	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = strings.TrimSpace(stderr.String())
	}

	return output, nil
}

// GetNamedContainerName gets the container name for a named config
func GetNamedContainerName(configName string) string {
	return SingBoxContainerPrefix + configName
}

// CreateAndStartNamedContainer creates and starts a named sing-box container
func (d *DockerService) CreateAndStartNamedContainer(configName string, hostConfigDir string) (string, error) {
	containerName := GetNamedContainerName(configName)

	// First try to remove any container with the same name
	_ = d.RemoveNamedContainer(configName)

	hostSingboxDir, err := resolveHostConfigDir(hostConfigDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve host config dir: %w", err)
	}
	log.Printf("Creating named container %s, mounting %s to /etc/sing-box", containerName, hostSingboxDir)

	// Ensure ACME data directory exists (using container-internal path)
	internalAcmeDir := filepath.Join(hostConfigDir, "acme")
	if err := os.MkdirAll(internalAcmeDir, 0755); err != nil {
		log.Printf("Warning: failed to create ACME directory %s: %v", internalAcmeDir, err)
	}
	hostAcmeDir := hostSingboxDir + "/acme"

	// Container config
	config := &container.Config{
		Image: SingBoxImageName,
		Cmd:   []string{"-D", ContainerDataDir, "-C", ContainerConfigDir + "/", "run"},
	}

	// Host config
	hostConfig := &container.HostConfig{
		NetworkMode: "host",
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   hostSingboxDir,
				Target:   "/etc/sing-box",
				ReadOnly: true,
			},
			{
				// ACME data directory (for automatic certificate requests)
				Type:     mount.TypeBind,
				Source:   hostAcmeDir,
				Target:   "/var/lib/sing-box/acme",
				ReadOnly: false,
			},
		},
		CapAdd: []string{"NET_ADMIN"},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			Memory:   512 * 1024 * 1024,
			NanoCPUs: 1000000000,
		},
	}

	// Create container
	resp, err := d.cli.ContainerCreate(
		d.ctx,
		config,
		hostConfig,
		nil,
		nil,
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := d.cli.ContainerStart(d.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		_ = d.cli.ContainerRemove(d.ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("Named container %s started with ID: %s", containerName, resp.ID[:12])
	return resp.ID, nil
}

// StopNamedContainer stops a named sing-box container
func (d *DockerService) StopNamedContainer(configName string) error {
	containerName := GetNamedContainerName(configName)
	timeout := 10
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}

	if err := d.cli.ContainerStop(d.ctx, containerName, stopOptions); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	log.Printf("Named container %s stopped", containerName)
	return nil
}

// RemoveNamedContainer removes a named sing-box container
func (d *DockerService) RemoveNamedContainer(configName string) error {
	containerName := GetNamedContainerName(configName)
	if err := d.cli.ContainerRemove(d.ctx, containerName, types.ContainerRemoveOptions{
		Force: true,
	}); err != nil {
		if !strings.Contains(err.Error(), "No such container") {
			return fmt.Errorf("failed to remove container: %w", err)
		}
	}

	log.Printf("Named container %s removed", containerName)
	return nil
}

// GetNamedContainerStatus gets the named container status
func (d *DockerService) GetNamedContainerStatus(configName string) (running bool, containerID string, err error) {
	containerName := GetNamedContainerName(configName)
	containers, err := d.cli.ContainerList(d.ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return false, "", fmt.Errorf("failed to list containers: %w", err)
	}

	// Manually filter container names because Docker API's name filter is substring matching
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName {
				return c.State == "running", c.ID, nil
			}
		}
	}

	return false, "", nil
}

// GetNamedContainerLogs gets the named container logs
func (d *DockerService) GetNamedContainerLogs(configName string, tail string) (string, error) {
	containerName := GetNamedContainerName(configName)
	if tail == "" {
		tail = "100"
	}

	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	}

	reader, err := d.cli.ContainerLogs(d.ctx, containerName, options)
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

// ListAllSingboxContainers lists all sing-box containers
func (d *DockerService) ListAllSingboxContainers() ([]ContainerInfo, error) {
	containers, err := d.cli.ContainerList(d.ctx, types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", SingBoxContainerPrefix)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
			name = strings.TrimPrefix(name, SingBoxContainerPrefix)
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

// ContainerInfo container information
type ContainerInfo struct {
	Name        string `json:"name"`
	ContainerID string `json:"container_id"`
	State       string `json:"state"`
	Status      string `json:"status"`
	Created     int64  `json:"created"`
}
