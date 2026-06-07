package tunnelrunner

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"singbox-config-service/internal/pkg/config"
)

type nativeRunner struct {
	binaryPath string
}

func NewRunner(cfg *config.Config) Runner {
	binaryPath := cfg.GetSingboxBinPath()
	if binaryPath == "" {
		if p, err := exec.LookPath("sing-box"); err == nil {
			binaryPath = p
		}
	}
	return &nativeRunner{binaryPath: binaryPath}
}

type tempInstance struct {
	cmd    *exec.Cmd
	logBuf *bytes.Buffer
}

var instances = make(map[string]*tempInstance)

func (n *nativeRunner) StartTemp(ctx context.Context, configPath string) (string, error) {
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

func (n *nativeRunner) StopTemp(ctx context.Context, id string) error {
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

func (n *nativeRunner) WaitTempReady(ctx context.Context, id string, port int, timeout time.Duration) error {
	return waitProxyReady(ctx, port, timeout)
}

func (n *nativeRunner) GetTempLogs(ctx context.Context, id string) string {
	inst, ok := instances[id]
	if !ok {
		return "instance not found"
	}
	return inst.logBuf.String()
}

func mustParsePid(id string) int {
	s := strings.TrimPrefix(id, "pid:")
	pid, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return pid
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
