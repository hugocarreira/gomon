//go:build !windows

package builder

import (
	"os/exec"
	"syscall"
)

func configureManagedProcess(cmd *exec.Cmd) bool {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return true
}

func terminateProcess(cmd *exec.Cmd, processGroup bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if processGroup {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}

func forceKillProcess(cmd *exec.Cmd, processGroup bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if processGroup {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return cmd.Process.Kill()
}
