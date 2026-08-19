package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestShouldIgnoreDir(t *testing.T) {
	if !ShouldIgnoreDir(".git") {
		t.Fatalf("expected .git to be ignored")
	}

	if ShouldIgnoreDir("normal") {
		t.Fatalf("did not expect normal directory to be ignored")
	}
}

func TestShouldWatchFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "main.go", want: true},
		{path: "nested/pkg/file.go", want: true},
		{path: "go.mod", want: true},
		{path: "go.sum", want: true},
		{path: "go.work", want: true},
		{path: "go.work.sum", want: true},
		{path: "README.md", want: false},
		{path: "main.go.txt", want: false},
	}
	for _, test := range tests {
		if got := ShouldWatchFile(test.path); got != test.want {
			t.Fatalf("ShouldWatchFile(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestDefineProjectPathWithFlag(t *testing.T) {
	got, err := DefineProjectPathWithFlag("/tmp/foo", []string{"cmd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/tmp/foo" {
		t.Fatalf("expected /tmp/foo, got %s", got)
	}
}

func TestDefineProjectPathWithArgs(t *testing.T) {
	got, err := DefineProjectPathWithFlag("", []string{"cmd", "/tmp/bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/tmp/bar" {
		t.Fatalf("expected /tmp/bar, got %s", got)
	}
}

func TestDefineProjectPathWithFlagDefaultsToWorkingDirectory(t *testing.T) {
	got, err := DefineProjectPathWithFlag("", []string{"cmd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDefineProjectPathDefaultsToWorkingDirectory(t *testing.T) {
	got, err := DefineProjectPath([]string{"cmd"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	if got, err := DefineProjectPath([]string{"cmd", "./project"}); err != nil || got != "project" {
		t.Fatalf("unexpected positional path: %q, %v", got, err)
	}
}

func TestVerifyProjectPath(t *testing.T) {
	dir := t.TempDir()
	if err := VerifyProjectPath(dir); err != nil {
		t.Fatalf("expected VerifyProjectPath to succeed for %s: %v", dir, err)
	}

	if err := VerifyProjectPath(filepath.Join(dir, "missing")); err == nil {
		t.Fatalf("expected VerifyProjectPath to fail for missing path")
	}

	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := VerifyProjectPath(file); err == nil {
		t.Fatal("expected VerifyProjectPath to reject a regular file")
	}
}

func TestHandleFilesAddsDir(t *testing.T) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("close watcher: %v", err)
		}
	})

	dir := t.TempDir()
	sub := filepath.Join(dir, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	f := NewFilesHandler(w)
	if err := f.HandleFiles(dir); err != nil {
		t.Fatalf("HandleFiles returned error: %v", err)
	}
	if !f.WasWatchedDir(dir) {
		t.Fatalf("expected %s to be recorded as watched", dir)
	}
}

func TestHandleFilesIgnoresMissingPath(t *testing.T) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Errorf("close watcher: %v", err)
		}
	})

	if err := NewFilesHandler(w).HandleFiles(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("expected missing event path to be ignored, got %v", err)
	}
}

func TestHandleFilesRejectsNilWatcherForDirectory(t *testing.T) {
	if err := NewFilesHandler(nil).HandleFiles(t.TempDir()); err == nil {
		t.Fatal("expected nil watcher to return an error")
	}
}
