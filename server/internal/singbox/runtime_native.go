// Package singbox provides services for managing sing-box configuration and containers.

package singbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"singbox-config-service/internal/pkg/config"
)

const (
	runsDir   = "singbox/run"
	pidSuffix = ".pid"
	logSuffix = ".log"
)

// NativeRuntime manages sing-box instances as OS processes.
type NativeRuntime struct {
	binaryPath string
	dataDir    string
}

// NewRuntime creates a Runtime backed by native OS processes.
func NewRuntime(cfg *config.Config) (Runtime, error) {
	binaryPath := cfg.GetSingboxBinPath()
	if binaryPath == "" {
		var err error
		binaryPath, err = exec.LookPath("sing-box")
		if err != nil {
			return nil, fmt.Errorf("sing-box binary not found: " +
				"set --singbox-bin or install sing-box in PATH")
		}
	}
	return &NativeRuntime{
		binaryPath: binaryPath,
		dataDir:    cfg.GetDataDir(),
	}, nil
}

func (n *NativeRuntime) runDir() string {
	return filepath.Join(n.dataDir, runsDir)
}

func (n *NativeRuntime) pidFile(name string) string {
	return filepath.Join(n.runDir(), name+pidSuffix)
}

func (n *NativeRuntime) logFile(name string) string {
	return filepath.Join(n.runDir(), name+logSuffix)
}

func (n *NativeRuntime) Start(ctx context.Context, name, configPath string) (string, error) {
	if err := os.MkdirAll(n.runDir(), 0755); err != nil {
		return "", fmt.Errorf("create run dir: %w", err)
	}

	// Check if already running
	if running, id, _ := n.checkRunning(name); running {
		return id, fmt.Errorf("instance %s is already running", name)
	}

	logPath := n.logFile(name)
	logF, err := os.Create(logPath)
	if err != nil {
		return "", fmt.Errorf("create log file: %w", err)
	}
	defer logF.Close()

	cmd := exec.CommandContext(ctx, n.binaryPath, "run", "-c", configPath)
	cmd.Stdout = logF
	cmd.Stderr = logF

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start sing-box: %w", err)
	}

	// Write PID file
	pid := cmd.Process.Pid
	pidStr := strconv.Itoa(pid)
	if err := os.WriteFile(n.pidFile(name), []byte(pidStr+"\n"), 0644); err != nil {
		_, _ = fmt.Fprintf(logF, "warning: failed to write pid file: %v\n", err)
	}

	return fmt.Sprintf("pid:%d", pid), nil
}

func (n *NativeRuntime) Stop(ctx context.Context, name string, timeout *int) error {
	running, pidStr, err := n.checkRunning(name)
	if err != nil {
		return err
	}
	if !running {
		return nil
	}

	pid := parsePID(pidStr)
	proc, err := os.FindProcess(pid)
	if err != nil {
		return n.cleanupPID(name)
	}

	// SIGTERM
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return n.cleanupPID(name)
	}

	// Wait for process to exit
	t := 10
	if timeout != nil {
		t = *timeout
	}
	waitCh := make(chan bool, 1)
	go func() {
		_, _ = proc.Wait()
		waitCh <- true
	}()
	select {
	case <-waitCh:
		// Graceful exit
	case <-time.After(time.Duration(t) * time.Second):
		_ = proc.Kill()
	case <-ctx.Done():
		_ = proc.Kill()
	}

	return n.cleanupPID(name)
}

func (n *NativeRuntime) Status(ctx context.Context, name string) (bool, string, error) {
	return n.checkRunning(name)
}

func (n *NativeRuntime) Logs(ctx context.Context, name, tail string) (string, error) {
	logPath := n.logFile(name)
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if tail == "" {
		return string(data), nil
	}
	tailN, err := strconv.Atoi(tail)
	if err != nil || tailN <= 0 || tailN >= len(strings.Split(string(data), "\n")) {
		return string(data), nil
	}
	lines := strings.Split(string(data), "\n")
	return strings.Join(lines[len(lines)-tailN:], "\n"), nil
}

func (n *NativeRuntime) Version(_ context.Context) (string, error) {
	cmd := exec.Command(n.binaryPath, "version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown", fmt.Errorf("get version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (n *NativeRuntime) List(ctx context.Context) ([]InstanceInfo, error) {
	entries, err := os.ReadDir(n.runDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	seen := make(map[string]bool)
	var result []InstanceInfo
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), pidSuffix)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		running, id, _ := n.checkRunning(name)
		state := "stopped"
		if running {
			state = "running"
		}
		result = append(result, InstanceInfo{
			Name:    name,
			ID:      id,
			Running: running,
			State:   state,
		})
	}
	return result, nil
}

func (n *NativeRuntime) Close() error {
	return nil
}

// checkRunning checks whether a process identified by name is running.
func (n *NativeRuntime) checkRunning(name string) (bool, string, error) {
	data, err := os.ReadFile(n.pidFile(name))
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false, "", n.cleanupPID(name)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, "", n.cleanupPID(name)
	}
	// Signal 0 checks existence without sending a signal
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false, "", n.cleanupPID(name)
	}
	return true, fmt.Sprintf("pid:%d", pid), nil
}

// cleanupPID removes the PID file for an instance.
func (n *NativeRuntime) cleanupPID(name string) error {
	if err := os.Remove(n.pidFile(name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// parsePID extracts the numeric PID from a "pid:<N>" string.
func parsePID(s string) int {
	pid, err := strconv.Atoi(strings.TrimPrefix(s, "pid:"))
	if err != nil {
		return -1
	}
	return pid
}
