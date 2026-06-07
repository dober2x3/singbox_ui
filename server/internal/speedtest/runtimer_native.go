

package speedtest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"singbox-config-service/internal/pkg/config"
)

// NativeTempRuntime creates temporary sing-box processes for speed tests.
type NativeTempRuntime struct {
	binaryPath string
}

// NewTempRuntime creates a TempRuntime backed by native OS processes.
func NewTempRuntime(cfg *config.AppConfig) TempRuntime {
	binaryPath := cfg.GetSingboxBinPath()
	if binaryPath == "" {
		if p, err := exec.LookPath("sing-box"); err == nil {
			binaryPath = p
		}
	}
	return &NativeTempRuntime{binaryPath: binaryPath}
}

type tempInstance struct {
	cmd    *exec.Cmd
	logBuf *bytes.Buffer
}

var instances = make(map[string]*tempInstance)

func (n *NativeTempRuntime) StartTemp(ctx context.Context, configPath string) (string, error) {
	if n.binaryPath == "" {
		return "", fmt.Errorf("sing-box binary not configured")
	}

	var logBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, n.binaryPath, "run", "-c", configPath)
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start: %w", err)
	}

	pid := cmd.Process.Pid
	id := fmt.Sprintf("pid:%d", pid)
	instances[id] = &tempInstance{cmd: cmd, logBuf: &logBuf}
	return id, nil
}

func (n *NativeTempRuntime) StopTemp(ctx context.Context, id string) error {
	inst, ok := instances[id]
	if !ok {
		return nil
	}
	pid := mustParsePid(id)
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
		_ = proc.Kill()
		_ = inst.cmd.Wait()
	}
	delete(instances, id)
	return nil
}

func (n *NativeTempRuntime) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	return waitProxyReady(ctx, port, timeout)
}

func (n *NativeTempRuntime) GetTempLogs(ctx context.Context, id string) string {
	inst, ok := instances[id]
	if !ok {
		return "instance not found"
	}
	return inst.logBuf.String()
}

// mustParsePid extracts a numeric PID from a "pid:N" string.
func mustParsePid(id string) int {
	s := strings.TrimPrefix(id, "pid:")
	pid, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return pid
}
