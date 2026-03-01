package builder

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

type IBuilder interface {
	BuildProject() error
	RunBinary() (*exec.Cmd, error)
	RestartBinary() error
	KillProcess(cmd *exec.Cmd) error
}

type Builder struct {
	projectDir string
	binaryPath string
	process    *exec.Cmd
}

func NewBuilder(projectDir, binaryPath string) IBuilder {
	return &Builder{
		projectDir: projectDir,
		binaryPath: binaryPath,
	}
}

func (b *Builder) BuildProject() error {
	cmd := exec.Command("go", "build", "-C", b.projectDir, "-o", b.binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *Builder) RunBinary() (*exec.Cmd, error) {
	cmd := exec.Command(b.binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func (b *Builder) RestartBinary() error {
	err := b.BuildProject()
	if err != nil {
		return err
	}

	err = b.KillProcess(b.process)
	if err != nil {
		return err
	}

	b.process, err = b.RunBinary()
	if err != nil {
		return err
	}

	return nil
}

func (b *Builder) KillProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	// Try graceful kill first (SIGTERM)
	err := cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		// Process may have already exited
		return nil
	}

	// Wait for process to exit with a timeout
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		// Force kill if it doesn't exit gracefully
		return cmd.Process.Kill()
	}
}
