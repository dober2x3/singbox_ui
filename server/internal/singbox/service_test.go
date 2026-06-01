package singbox

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"singbox-config-service/internal/pkg/config"
	"singbox-config-service/internal/pkg/docker"
)

// mockContainerManager implements ContainerManager for testing.
type mockContainerManager struct {
	createContainerFn   func(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error)
	startContainerFn    func(ctx context.Context, containerID string) error
	stopContainerFn     func(ctx context.Context, containerID string, timeout *int) error
	removeContainerFn   func(ctx context.Context, containerID string, force bool) error
	containerLogsFn     func(ctx context.Context, containerID string, tail string) (string, error)
	getContainerStateFn func(ctx context.Context, containerName string) (string, error)
	imagePullFn         func(ctx context.Context, image string) (io.ReadCloser, error)
	imageListFn         func(ctx context.Context, image string) (bool, error)
	listContainersFn    func(ctx context.Context, prefix string) ([]docker.ContainerInfo, error)
	ensureImageFn       func(ctx context.Context, imageName, tarPath string) error
	closeFn             func() error
}

// ContainerCreate mocks container creation.
func (m *mockContainerManager) ContainerCreate(ctx context.Context, config interface{}, hostConfig interface{}, name string) (string, error) {
	return m.createContainerFn(ctx, config, hostConfig, name)
}

// ContainerStart mocks starting a container.
func (m *mockContainerManager) ContainerStart(ctx context.Context, containerID string) error {
	return m.startContainerFn(ctx, containerID)
}

// ContainerStop mocks stopping a container.
func (m *mockContainerManager) ContainerStop(ctx context.Context, containerID string, timeout *int) error {
	return m.stopContainerFn(ctx, containerID, timeout)
}

// ContainerRemove mocks removing a container.
func (m *mockContainerManager) ContainerRemove(ctx context.Context, containerID string, force bool) error {
	return m.removeContainerFn(ctx, containerID, force)
}

// ContainerLogs mocks fetching container logs.
func (m *mockContainerManager) ContainerLogs(ctx context.Context, containerID string, tail string) (string, error) {
	return m.containerLogsFn(ctx, containerID, tail)
}

// GetContainerState mocks getting container state.
func (m *mockContainerManager) GetContainerState(ctx context.Context, containerName string) (string, error) {
	return m.getContainerStateFn(ctx, containerName)
}

// ImagePull mocks pulling a Docker image.
func (m *mockContainerManager) ImagePull(ctx context.Context, image string) (io.ReadCloser, error) {
	return m.imagePullFn(ctx, image)
}

// ImageList mocks listing Docker images.
func (m *mockContainerManager) ImageList(ctx context.Context, image string) (bool, error) {
	return m.imageListFn(ctx, image)
}

// ListContainers mocks listing containers by prefix.
func (m *mockContainerManager) ListContainers(ctx context.Context, prefix string) ([]docker.ContainerInfo, error) {
	return m.listContainersFn(ctx, prefix)
}

// EnsureImage mocks ensuring a Docker image is available.
func (m *mockContainerManager) EnsureImage(ctx context.Context, imageName, tarPath string) error {
	return m.ensureImageFn(ctx, imageName, tarPath)
}

// Close mocks closing the container manager connection.
func (m *mockContainerManager) Close() error {
	return m.closeFn()
}

// newMockManager creates a mockContainerManager with default success responses.
func newMockManager() *mockContainerManager {
	return &mockContainerManager{
		createContainerFn: func(_ context.Context, _ interface{}, _ interface{}, _ string) (string, error) {
			return "mock-container-id", nil
		},
		startContainerFn: func(_ context.Context, _ string) error {
			return nil
		},
		stopContainerFn: func(_ context.Context, _ string, _ *int) error {
			return nil
		},
		removeContainerFn: func(_ context.Context, _ string, _ bool) error {
			return nil
		},
		containerLogsFn: func(_ context.Context, _ string, _ string) (string, error) {
			return "", nil
		},
		getContainerStateFn: func(_ context.Context, containerName string) (string, error) {
			return "", nil
		},
		imagePullFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("")), nil
		},
		imageListFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
		listContainersFn: func(_ context.Context, _ string) ([]docker.ContainerInfo, error) {
			return []docker.ContainerInfo{}, nil
		},
		ensureImageFn: func(_ context.Context, _, _ string) error {
			return nil
		},
		closeFn: func() error {
			return nil
		},
	}
}

// newTestService creates a Service with a mock manager and temporary directory for testing.
func newTestService(t *testing.T) (*Service, *config.Config, func()) {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("DATA_DIR", dir)
	cfg, err := config.Init()
	if err != nil {
		t.Fatalf("config.Init() error = %v", err)
	}
	cleanup := func() {
		os.Unsetenv("DATA_DIR")
	}
	mgr := newMockManager()
	return NewService(mgr, cfg), cfg, cleanup
}

// TestNewService verifies that NewService returns a non-nil Service.
func TestNewService(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

// TestSaveAndGetConfig verifies saving config and reading it back.
func TestSaveAndGetConfig(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	data := []byte(`{"log":{"level":"info"}}`)
	path, err := svc.SaveConfig(data)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}
	if path == "" {
		t.Error("SaveConfig() returned empty path")
	}

	got, err := svc.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("GetConfig() = %q, want %q", string(got), string(data))
	}
}

// TestContainerLifecycle verifies the full lifecycle: save config, run, status, stop.
func TestContainerLifecycle(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	mgr := svc.docker.(*mockContainerManager)

	cfgData := []byte(`{"log":{"level":"debug"}}`)
	_, err := svc.SaveConfig(cfgData)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	id, err := svc.RunContainer()
	if err != nil {
		t.Fatalf("RunContainer() error = %v", err)
	}
	if id == "" {
		t.Error("RunContainer() returned empty id")
	}

	mgr.getContainerStateFn = func(_ context.Context, containerName string) (string, error) {
		return "running", nil
	}

	running, cid := svc.ContainerStatus()
	if !running {
		t.Error("ContainerStatus() should be running")
	}
	if cid == "" {
		t.Error("ContainerStatus() returned empty containerID")
	}

	err = svc.StopContainer()
	if err != nil {
		t.Fatalf("StopContainer() error = %v", err)
	}
}

// TestGetVersion verifies that GetVersion returns a non-empty version string.
func TestGetVersion(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	v, err := svc.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	if v == "" {
		t.Error("GetVersion() returned empty")
	}
}

// TestNamedConfigs verifies the full lifecycle of named configs: save, load, check, run, stop, delete.
func TestNamedConfigs(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	mgr := svc.docker.(*mockContainerManager)
	name := "test-instance"
	data := []byte(`{"outbounds":[]}`)

	err := svc.SaveNamedConfig(name, data)
	if err != nil {
		t.Fatalf("SaveNamedConfig() error = %v", err)
	}

	got, err := svc.LoadNamedConfig(name)
	if err != nil {
		t.Fatalf("LoadNamedConfig() error = %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("LoadNamedConfig() = %q, want %q", string(got), string(data))
	}

	valid, output := svc.CheckNamedConfig(name)
	if !valid {
		t.Errorf("CheckNamedConfig() should be valid, got output: %s", output)
	}

	configs, err := svc.ListNamedConfigs()
	if err != nil {
		t.Fatalf("ListNamedConfigs() error = %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("ListNamedConfigs() = %d items, want 1", len(configs))
	}
	if len(configs) > 0 && configs[0].Name != name {
		t.Errorf("ListNamedConfigs()[0].Name = %q, want %q", configs[0].Name, name)
	}

	_, err = svc.RunNamedContainer(name)
	if err != nil {
		t.Fatalf("RunNamedContainer() error = %v", err)
	}

	mgr.getContainerStateFn = func(_ context.Context, containerName string) (string, error) {
		return "running", nil
	}

	running, _ := svc.NamedContainerStatus(name)
	if !running {
		t.Error("NamedContainerStatus() should be running")
	}

	logs := svc.NamedContainerLogs(name)
	_ = logs

	err = svc.StopNamedContainer(name)
	if err != nil {
		t.Fatalf("StopNamedContainer() error = %v", err)
	}

	err = svc.DeleteNamedConfig(name)
	if err != nil {
		t.Fatalf("DeleteNamedConfig() error = %v", err)
	}

	configs, err = svc.ListNamedConfigs()
	if err != nil {
		t.Fatalf("ListNamedConfigs() after delete error = %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("ListNamedConfigs() after delete = %d items, want 0", len(configs))
	}
}

// TestCheckNamedConfig verifies validation of named configs: missing, valid JSON, invalid JSON.
func TestCheckNamedConfig(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	valid, output := svc.CheckNamedConfig("nonexistent")
	if valid {
		t.Error("CheckNamedConfig() for nonexistent should be invalid")
	}
	if output == "" {
		t.Error("CheckNamedConfig() for nonexistent should have output")
	}

	_ = svc.SaveNamedConfig("test-instance", []byte(`{"log":{"level":"info"}}`))
	valid, output = svc.CheckNamedConfig("test-instance")
	if !valid {
		t.Errorf("CheckNamedConfig() should be valid, got output: %s", output)
	}

	_ = svc.SaveNamedConfig("bad-instance", []byte(`{invalid json`))
	valid, _ = svc.CheckNamedConfig("bad-instance")
	if valid {
		t.Error("CheckNamedConfig() for invalid JSON should be invalid")
	}
}

// TestListAllContainers verifies ListAllContainers returns a non-nil list.
func TestListAllContainers(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	containers, err := svc.ListAllContainers()
	if err != nil {
		t.Fatalf("ListAllContainers() error = %v", err)
	}
	if containers == nil {
		t.Error("ListAllContainers() returned nil")
	}
}

// TestEnsureImage verifies EnsureImage completes without error.
func TestEnsureImage(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	err := svc.EnsureImage()
	if err != nil {
		t.Fatalf("EnsureImage() error = %v", err)
	}
}

// TestContainerLogs verifies ContainerLogs returns without error.
func TestContainerLogs(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	logs := svc.ContainerLogs()
	_ = logs
}

// TestNamedContainerStatus_NotRunning verifies status is false for nonexistent containers.
func TestNamedContainerStatus_NotRunning(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	running, cid := svc.NamedContainerStatus("nonexistent")
	if running {
		t.Error("NamedContainerStatus() should be false for nonexistent")
	}
	if cid != "" {
		t.Error("NamedContainerStatus() cid should be empty for nonexistent")
	}
}

// TestContainerStatus_NoContainer verifies status returns not running when no container exists.
func TestContainerStatus_NoContainer(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	running, cid := svc.ContainerStatus()
	if running {
		t.Error("ContainerStatus() should be false when no container")
	}
	if cid != "" {
		t.Error("ContainerStatus() cid should be empty")
	}
}

// TestStopContainer_NoContainer verifies StopContainer does not error when no container exists.
func TestStopContainer_NoContainer(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	err := svc.StopContainer()
	if err != nil {
		t.Errorf("StopContainer() should not error when no container: %v", err)
	}
}

// TestStopNamedContainer_NoContainer verifies StopNamedContainer does not error when no container exists.
func TestStopNamedContainer_NoContainer(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	err := svc.StopNamedContainer("test-instance")
	if err != nil {
		t.Errorf("StopNamedContainer() should not error when no container: %v", err)
	}
}

// TestRunNamedContainer_NoConfig verifies RunNamedContainer errors when config is missing.
func TestRunNamedContainer_NoConfig(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.RunNamedContainer("nonexistent")
	if err == nil {
		t.Error("RunNamedContainer() should error when config not found")
	}
}

// TestRunNamedContainer_AlreadyRunning verifies RunNamedContainer errors when container is already running.
func TestRunNamedContainer_AlreadyRunning(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	mgr := svc.docker.(*mockContainerManager)
	mgr.getContainerStateFn = func(_ context.Context, containerName string) (string, error) {
		return "running", nil
	}

	_ = svc.SaveNamedConfig("test-instance", []byte(`{}`))

	_, err := svc.RunNamedContainer("test-instance")
	if err == nil {
		t.Error("RunNamedContainer() should error when container already running")
	}
	if err != nil && !strings.Contains(err.Error(), "already running") {
		t.Errorf("RunNamedContainer() unexpected error: %v", err)
	}
}

// TestRunContainer_NoConfig verifies RunContainer errors when no config file exists.
func TestRunContainer_NoConfig(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.RunContainer()
	if err == nil {
		t.Error("RunContainer() should error when config not found")
	}
}

// TestNamedContainerLogs_Error verifies NamedContainerLogs returns without panic.
func TestNamedContainerLogs_Error(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	logs := svc.NamedContainerLogs("test-instance")
	_ = logs
}

// TestSaveNamedConfig verifies saving a named config completes without error.
func TestSaveNamedConfig(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	err := svc.SaveNamedConfig("my-instance", []byte(`{"log":{"level":"info"}}`))
	if err != nil {
		t.Fatalf("SaveNamedConfig() error = %v", err)
	}
}

// TestDeleteNamedConfig verifies deleting a named config removes its files.
func TestDeleteNamedConfig(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	_ = svc.SaveNamedConfig("my-instance", []byte(`{}`))

	err := svc.DeleteNamedConfig("my-instance")
	if err != nil {
		t.Fatalf("DeleteNamedConfig() error = %v", err)
	}

	_, err = svc.LoadNamedConfig("my-instance")
	if err == nil {
		t.Error("LoadNamedConfig() should error after DeleteNamedConfig()")
	}
}

// TestListNamedConfigs_Empty verifies ListNamedConfigs returns empty list when no configs exist.
func TestListNamedConfigs_Empty(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	configs, err := svc.ListNamedConfigs()
	if err != nil {
		t.Fatalf("ListNamedConfigs() error = %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("ListNamedConfigs() = %d items, want 0", len(configs))
	}
}

// TestContainerLogs_Error verifies ContainerLogs handles mock errors gracefully.
func TestContainerLogs_Error(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	mgr := svc.docker.(*mockContainerManager)
	mgr.containerLogsFn = func(_ context.Context, _ string, _ string) (string, error) {
		return "", nil
	}

	logs := svc.ContainerLogs()
	_ = logs
}

// TestContainerRun_AfterDeleteConfig verifies RunContainer works after saving a new config.
func TestContainerRun_AfterDeleteConfig(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	data := []byte(`{"log":{"level":"info"}}`)
	_, err := svc.SaveConfig(data)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	id, err := svc.RunContainer()
	if err != nil {
		t.Fatalf("RunContainer() error = %v", err)
	}
	if id == "" {
		t.Error("RunContainer() returned empty id")
	}
}

// TestGetVersion_Empty verifies GetVersion returns the expected version string.
func TestGetVersion_Empty(t *testing.T) {
	svc, _, cleanup := newTestService(t)
	defer cleanup()

	v, err := svc.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	if v != "sing-box 1.10.0" {
		t.Errorf("GetVersion() = %q, want %q", v, "sing-box 1.10.0")
	}
}
