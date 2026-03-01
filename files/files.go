package files

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
		if name == pattern || strings.HasPrefix(name, "."+pattern) {
			return true
		}
	}
	return false
}

// ShouldWatchFile returns true if the file should trigger a rebuild.
func ShouldWatchFile(path string) bool {
	return strings.HasSuffix(path, ".go")
}

// Files handles file system operations for the watcher.
type Files struct {
	watcher *fsnotify.Watcher
}

// NewFilesHandler creates a new Files handler.
func NewFilesHandler(watcher *fsnotify.Watcher) *Files {
	return &Files{
		watcher: watcher,
	}
}

// HandleFiles adds a file or directory to the watcher.
func (f *Files) HandleFiles(path string) error {
	isDir, err := f.IsDir(path)
	if err == nil && isDir {
		err := f.AddDir(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddDir recursively adds a directory and its subdirectories to the watcher.
func (f *Files) AddDir(dir string) error {
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if ShouldIgnoreDir(info.Name()) {
				return filepath.SkipDir
			}
			err = f.watcher.Add(path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
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

	return args[1], nil
}

// DefineProjectPathWithFlag returns the project path from flag, positional argument, or current directory.
func DefineProjectPathWithFlag(flagPath string, args []string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}

	if len(args) >= 2 {
		return args[1], nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return cwd, nil
}

// VerifyProjectPath checks if the project path exists.
func VerifyProjectPath(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	return nil
}
