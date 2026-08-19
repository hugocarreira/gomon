package builder

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const processStopTimeout = 5 * time.Second

type IBuilder interface {
	BuildProject() error
	RunBinary() (*exec.Cmd, error)
	RestartBinary() error
	KillProcess(cmd *exec.Cmd) error
	Close() error
}

type runningProcess struct {
	cmd          *exec.Cmd
	done         <-chan error
	processGroup bool
}

type Builder struct {
	projectDir string
	binaryPath string
	outputPath string
	tempDir    string
	process    *exec.Cmd
	running    *runningProcess
	closed     bool
	mu         sync.Mutex
}

func NewBuilder(projectDir, binaryPath string) IBuilder {
	return &Builder{
		projectDir: projectDir,
		binaryPath: binaryPath,
	}
}

// BuildProject builds directly to the configured output path. Restarts use a
// staging path so a failed build never interrupts the currently running app.
func (b *Builder) BuildProject() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("builder is closed")
	}

	output, err := b.outputPathLocked()
	if err != nil {
		return err
	}
	return b.build(output)
}

func (b *Builder) RunBinary() (*exec.Cmd, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("builder is closed")
	}
	b.refreshProcessLocked()
	if b.running != nil {
		return nil, errors.New("binary is already running")
	}

	output, err := b.outputPathLocked()
	if err != nil {
		return nil, err
	}
	return b.startLocked(output)
}

func (b *Builder) RestartBinary() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errors.New("builder is closed")
	}

	output, err := b.outputPathLocked()
	if err != nil {
		return err
	}
	stage, err := b.newStagePath(output)
	if err != nil {
		return fmt.Errorf("create build staging path: %w", err)
	}
	defer os.Remove(stage)

	if err := b.build(stage); err != nil {
		return err
	}

	b.refreshProcessLocked()
	if b.running != nil {
		if err := b.stopRunningLocked(b.running); err != nil {
			return fmt.Errorf("stop previous process: %w", err)
		}
		b.running = nil
		b.process = nil
	}

	if err := replaceBinary(stage, output); err != nil {
		return fmt.Errorf("promote built binary: %w", err)
	}
	_, err = b.startLocked(output)
	return err
}

func (b *Builder) KillProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	b.mu.Lock()
	if b.running != nil && b.running.cmd == cmd {
		err := b.stopRunningLocked(b.running)
		b.running = nil
		b.process = nil
		b.mu.Unlock()
		return err
	}
	b.mu.Unlock()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	if err := terminateProcess(cmd, false); err != nil && !isProcessDone(err) {
		return err
	}
	return waitForExit(done, cmd, false)
}

func (b *Builder) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true

	var stopErr error
	if b.running != nil {
		stopErr = b.stopRunningLocked(b.running)
		b.running = nil
		b.process = nil
	}
	if b.tempDir != "" {
		if err := os.RemoveAll(b.tempDir); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	return stopErr
}

func (b *Builder) outputPathLocked() (string, error) {
	if b.outputPath != "" {
		return b.outputPath, nil
	}

	if b.binaryPath != "" {
		if filepath.IsAbs(b.binaryPath) {
			b.outputPath = filepath.Clean(b.binaryPath)
		} else {
			absolute, err := filepath.Abs(filepath.Join(b.projectDir, b.binaryPath))
			if err != nil {
				return "", fmt.Errorf("resolve binary path: %w", err)
			}
			b.outputPath = absolute
		}
		return b.outputPath, nil
	}

	tempDir, err := os.MkdirTemp("", "gomon-")
	if err != nil {
		return "", fmt.Errorf("create temporary binary directory: %w", err)
	}
	b.tempDir = tempDir
	name := "app"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	b.outputPath = filepath.Join(tempDir, name)
	return b.outputPath, nil
}

func (b *Builder) newStagePath(output string) (string, error) {
	file, err := os.CreateTemp(filepath.Dir(output), "."+filepath.Base(output)+".gomon-*")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", err
	}
	return path, nil
}

func (b *Builder) build(output string) error {
	cmd := exec.Command("go", "build", "-o", output, ".")
	cmd.Dir = b.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) startLocked(output string) (*exec.Cmd, error) {
	cmd := exec.Command(output)
	cmd.Dir = b.projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	processGroup := configureManagedProcess(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	running := &runningProcess{cmd: cmd, done: done, processGroup: processGroup}
	b.running = running
	b.process = cmd
	return cmd, nil
}

func (b *Builder) refreshProcessLocked() {
	if b.running == nil {
		return
	}
	select {
	case <-b.running.done:
		b.running = nil
		b.process = nil
	default:
	}
}

func (b *Builder) stopRunningLocked(running *runningProcess) error {
	if running == nil || running.cmd == nil || running.cmd.Process == nil {
		return nil
	}
	if err := terminateProcess(running.cmd, running.processGroup); err != nil && !isProcessDone(err) {
		return err
	}
	return waitForExit(running.done, running.cmd, running.processGroup)
}

func waitForExit(done <-chan error, cmd *exec.Cmd, processGroup bool) error {
	timer := time.NewTimer(processStopTimeout)
	defer timer.Stop()

	select {
	case err := <-done:
		return normalizeWaitError(err)
	case <-timer.C:
		if err := forceKillProcess(cmd, processGroup); err != nil && !isProcessDone(err) {
			return err
		}
		select {
		case err := <-done:
			return normalizeWaitError(err)
		case <-time.After(time.Second):
			return errors.New("process did not exit after force kill")
		}
	}
}

func normalizeWaitError(err error) error {
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return err
}

func isProcessDone(err error) bool {
	return errors.Is(err, os.ErrProcessDone)
}

func replaceBinary(stage, output string) error {
	if err := os.Remove(output); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(stage, output)
}
