package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// DefaultIgnorePatterns contains directory names to ignore when watching.
var DefaultIgnorePatterns = []string{
	".git",
	".svn",
	".hg",
	"vendor",
	"node_modules",
	".idea",
	".vscode",
}

// ShouldIgnoreDir returns true if the directory should be ignored.
func ShouldIgnoreDir(name string) bool {
	for _, pattern := range DefaultIgnorePatterns {
		if name == pattern {
			return true
		}
	}
	return false
}

// ShouldWatchFile returns true if the file is a Go build input.
func ShouldWatchFile(path string) bool {
	switch filepath.Base(path) {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return strings.HasSuffix(filepath.Base(path), ".go")
	}
}

// Files handles file system operations for the watcher.
type Files struct {
	watcher     *fsnotify.Watcher
	mu          sync.RWMutex
	watchedDirs map[string]struct{}
}

// NewFilesHandler creates a new Files handler.
func NewFilesHandler(watcher *fsnotify.Watcher) *Files {
	return &Files{
		watcher:     watcher,
		watchedDirs: make(map[string]struct{}),
	}
}

// HandleFiles adds a directory to the watcher. Missing paths are ignored so a
// remove/rename event that races with stat does not stop the watcher.
func (f *Files) HandleFiles(path string) error {
	isDir, err := f.IsDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !isDir {
		return nil
	}

	return f.AddDir(path)
}

// AddDir recursively adds a directory and its subdirectories to the watcher.
func (f *Files) AddDir(dir string) error {
	if f.watcher == nil {
		return fmt.Errorf("watcher is nil")
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve directory %q: %w", dir, err)
	}

	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", path, walkErr)
		}
		if entry.IsDir() && path != dir && ShouldIgnoreDir(entry.Name()) {
			return filepath.SkipDir
		}
		if !entry.IsDir() {
			return nil
		}

		if err := f.watcher.Add(path); err != nil {
			return fmt.Errorf("watch %q: %w", path, err)
		}
		f.mu.Lock()
		f.watchedDirs[path] = struct{}{}
		f.mu.Unlock()
		return nil
	})
	return err
}

// WasWatchedDir reports whether a path has been registered as a directory.
// The historical entry is retained so remove and rename events remain useful.
func (f *Files) WasWatchedDir(path string) bool {
	path, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	f.mu.RLock()
	_, ok := f.watchedDirs[path]
	f.mu.RUnlock()
	return ok
}

// IsDir returns true if the path is a directory.
func (f *Files) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

// DefineProjectPath returns the project path from positional argument or current directory.
func DefineProjectPath(args []string) (string, error) {
	if len(args) < 2 {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return cwd, nil
	}

	return filepath.Clean(args[1]), nil
}

// DefineProjectPathWithFlag returns the project path from flag, positional argument, or current directory.
func DefineProjectPathWithFlag(flagPath string, args []string) (string, error) {
	if flagPath != "" {
		return filepath.Clean(flagPath), nil
	}

	if len(args) >= 2 {
		return filepath.Clean(args[1]), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd, nil
}

// VerifyProjectPath checks that the project path exists and is a directory.
func VerifyProjectPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("project path %q is not a directory", path)
	}

	return nil
}
