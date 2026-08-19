//go:build !windows

package builder

import (
	"errors"
	"os/exec"
	"syscall"
)

type processControl struct {
	processGroup bool
}

func newProcessControl() (processControl, error) {
	return processControl{processGroup: true}, nil
}

func prepareManagedProcess(cmd *exec.Cmd, control processControl) {
	if control.processGroup {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
}

func attachProcessControl(_ *exec.Cmd, _ processControl) error {
	return nil
}

func terminateProcess(cmd *exec.Cmd, control processControl) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if control.processGroup {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}

func forceKillProcess(cmd *exec.Cmd, control processControl) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if control.processGroup {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return cmd.Process.Kill()
}

func closeProcessControl(_ processControl) error {
	return nil
}

func processAlreadyGone(err error) bool {
	return errors.Is(err, syscall.ESRCH)
}
