package builder

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestKillProcessNil(t *testing.T) {
	b := &Builder{}
	if err := b.KillProcess(nil); err != nil {
		t.Fatalf("expected nil when killing nil process, got %v", err)
	}
}

func TestKillProcessTerminatesCommand(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper command: %v", err)
	}

	b := &Builder{}
	if err := b.KillProcess(cmd); err != nil {
		t.Fatalf("KillProcess returned error: %v", err)
	}

	if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
		t.Fatalf("process still running after KillProcess")
	}
}
