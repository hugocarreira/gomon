//go:build windows

package builder

import "os/exec"

// Windows does not provide a portable SIGTERM equivalent through os.Process.
// Termination therefore uses Kill immediately; the command remains isolated
// behind this helper so a Job Object can be added without changing Builder.
func configureManagedProcess(cmd *exec.Cmd) bool {
	return false
}

func terminateProcess(cmd *exec.Cmd, _ bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

func forceKillProcess(cmd *exec.Cmd, _ bool) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
