package files

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type Files struct {
	watcher *fsnotify.Watcher
}

func NewFilesHandler(watcher *fsnotify.Watcher) *Files {
	return &Files{
		watcher: watcher,
	}
}

func (f *Files) HandleFiles(e string) error {
	isDir, err := f.IsDir(e)
	if err == nil && isDir {
		err := f.AddDir(e)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *Files) AddDir(dir string) error {
	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			err = f.watcher.Add(path)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (f *Files) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return info.IsDir(), nil
}

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

func VerifyProjectPath(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	return nil
}
