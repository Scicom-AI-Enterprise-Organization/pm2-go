package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/huseinzol05/pm2-go/daemon/internal/paths"
)

// EnsureRunning checks if the daemon is reachable; if not, it spawns a new
// daemon process and waits up to timeout for the socket to appear.
func EnsureRunning(ctx context.Context, timeout time.Duration) error {
	if SocketAlive() {
		return nil
	}
	if err := Spawn(); err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if SocketAlive() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return fmt.Errorf("daemon did not become ready within %s", timeout)
}

func SocketAlive() bool {
	c, err := net.DialTimeout("unix", paths.RPCSocket(), 200*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// Spawn re-execs the current binary with the hidden `__daemon` subcommand,
// detaching it from the current terminal.
func Spawn() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(paths.DaemonLog(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "__daemon")
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(), "PM2_GO_DAEMON_BOOT=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Detach: do not Wait. PID file is written by the child.
	go func() { _ = cmd.Process.Release() }()
	return nil
}

// WritePIDFile records the current process pid into daemon.pid.
func WritePIDFile() error {
	return os.WriteFile(paths.DaemonPid(), []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o644)
}

// RemovePIDFile deletes the daemon pid file (best effort).
func RemovePIDFile() { _ = os.Remove(paths.DaemonPid()) }
