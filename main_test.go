package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected version command to succeed, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "gomon version") {
		t.Fatalf("unexpected version output: %q", stdout.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected help command to succeed, got %d: %s", code, stderr.String())
	}
	for _, expected := range []string{"-config", "-log-level", "project_path"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("help output missing %q: %s", expected, stdout.String())
		}
	}
}

func TestRunRejectsAmbiguousProjectPaths(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-path", "/tmp/one", "/tmp/two"}, &stdout, &stderr); code != 2 {
		t.Fatalf("expected ambiguous path to return 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "either --path") {
		t.Fatalf("unexpected error: %s", stderr.String())
	}
}

func TestRunUsesPositionalArgumentAfterFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-debounce", "1", "/definitely/missing-project"}, &stdout, &stderr); code != 1 {
		t.Fatalf("expected missing project to return 1, got %d", code)
	}
	if strings.Contains(stderr.String(), "-debounce") {
		t.Fatalf("flag name was incorrectly treated as project path: %s", stderr.String())
	}
}

func TestRunStartsAndStopsProjectFromAnyWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-driven integration test uses Unix process signals")
	}
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	outside := t.TempDir()
	if err := os.Chdir(outside); err != nil {
		t.Fatalf("change to unrelated working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/run-fixture\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	mainSource := []byte(`package main

import (
	"os"
	"time"
)

func main() {
	_ = os.WriteFile("ready", []byte("ready"), 0o644)
	time.Sleep(30 * time.Second)
}
`)
	if err := os.WriteFile(filepath.Join(project, "main.go"), mainSource, 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	var stdout, stderr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- run([]string{"--path", project, "--binary", filepath.Join(project, "app"), "--debounce", "10", "--log-level", "error"}, &stdout, &stderr)
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(project, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("project process did not signal readiness")
		}
		time.Sleep(10 * time.Millisecond)
	}
	terminateCurrentProcess(t)
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("run returned %d, stderr: %s", code, stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not stop after termination signal")
	}
}
