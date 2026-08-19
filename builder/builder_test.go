package builder

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
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

func TestRestartBuildKeepsPreviousProcessOnBuildError(t *testing.T) {
	project := writeBuilderProject(t, `package main

import "time"

func main() { for { time.Sleep(time.Second) } }
`)

	instance := NewBuilder(project, "")
	b := instance.(*Builder)
	t.Cleanup(func() { _ = b.Close() })

	if err := instance.RestartBinary(); err != nil {
		t.Fatalf("initial restart failed: %v", err)
	}
	first := b.process
	if first == nil {
		t.Fatal("expected initial process to be running")
	}

	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\nfunc main() {"), 0o644); err != nil {
		t.Fatalf("write invalid source: %v", err)
	}
	if err := instance.RestartBinary(); err == nil {
		t.Fatal("expected invalid source to fail the restart")
	}
	if b.process != first {
		t.Fatal("expected the previous process to remain active after a failed build")
	}
}

func TestBuilderRunsBinaryFromProjectDirectory(t *testing.T) {
	project := writeBuilderProject(t, `package main

import (
	"os"
	"time"
)

func main() {
	cwd, _ := os.Getwd()
	_ = os.WriteFile("cwd.txt", []byte(cwd), 0o644)
	time.Sleep(30 * time.Second)
}
`)

	instance := NewBuilder(project, "")
	b := instance.(*Builder)
	t.Cleanup(func() { _ = b.Close() })
	if err := instance.RestartBinary(); err != nil {
		t.Fatalf("restart failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(filepath.Join(project, "cwd.txt"))
		if err == nil {
			if string(contents) != project {
				t.Fatalf("expected child cwd %q, got %q", project, string(contents))
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("child did not write its working directory")
}

func TestBuildProjectAndRunBinary(t *testing.T) {
	project := writeBuilderProject(t, `package main

import "time"

func main() { time.Sleep(30 * time.Second) }
`)

	instance := NewBuilder(project, "")
	b := instance.(*Builder)
	t.Cleanup(func() { _ = b.Close() })
	if err := instance.BuildProject(); err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if _, err := os.Stat(b.outputPath); err != nil {
		t.Fatalf("expected built binary %q: %v", b.outputPath, err)
	}
	cmd, err := instance.RunBinary()
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if err := instance.KillProcess(cmd); err != nil {
		t.Fatalf("kill failed: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestBuilderRejectsMissingOutputDirectory(t *testing.T) {
	project := writeBuilderProject(t, "package main\nfunc main() {}\n")
	instance := NewBuilder(project, filepath.Join("missing", "app"))
	if err := instance.RestartBinary(); err == nil {
		t.Fatal("expected staging path creation to fail")
	}
}

func TestBuilderRunWithoutBuiltBinaryFails(t *testing.T) {
	project := writeBuilderProject(t, "package main\nfunc main() {}\n")
	instance := NewBuilder(project, "missing-app")
	b := instance.(*Builder)
	if _, err := instance.RunBinary(); err == nil {
		t.Fatal("expected running a missing binary to fail")
	}
	if err := b.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestProcessHelpersHandleCompletedAndNilProcesses(t *testing.T) {
	if !isProcessDone(os.ErrProcessDone) {
		t.Fatal("expected os.ErrProcessDone to be recognized")
	}
	if forceKillProcess(nil, processControl{}) != nil {
		t.Fatal("expected nil process force kill to be harmless")
	}
}

func writeBuilderProject(t *testing.T, mainSource string) string {
	t.Helper()
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/gomon-fixture\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte(mainSource), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	return project
}
