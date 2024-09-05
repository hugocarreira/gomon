package builder

import (
	"os"
	"os/exec"
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
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}

	return nil
}
