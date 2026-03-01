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
	if !ShouldWatchFile("main.go") {
		t.Fatalf("expected .go files to be watched")
	}

	if ShouldWatchFile("README.md") {
		t.Fatalf("did not expect non-.go files to be watched")
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

func TestVerifyProjectPath(t *testing.T) {
	dir := t.TempDir()
	if err := VerifyProjectPath(dir); err != nil {
		t.Fatalf("expected VerifyProjectPath to succeed for %s: %v", dir, err)
	}

	if err := VerifyProjectPath(filepath.Join(dir, "missing")); err == nil {
		t.Fatalf("expected VerifyProjectPath to fail for missing path")
	}
}

func TestHandleFilesAddsDir(t *testing.T) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Close()

	dir := t.TempDir()
	sub := filepath.Join(dir, "src")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	f := NewFilesHandler(w)
	if err := f.HandleFiles(dir); err != nil {
		t.Fatalf("HandleFiles returned error: %v", err)
	}
}
