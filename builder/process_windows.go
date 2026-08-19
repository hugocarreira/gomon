//go:build windows

package builder

import (
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processControl struct {
	job windows.Handle
}

func newProcessControl() (processControl, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return processControl{}, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		_ = windows.CloseHandle(job)
		return processControl{}, err
	}
	return processControl{job: job}, nil
}

func prepareManagedProcess(_ *exec.Cmd, _ processControl) {}

func attachProcessControl(cmd *exec.Cmd, control processControl) error {
	if control.job == 0 {
		return nil
	}
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,
		false,
		uint32(cmd.Process.Pid),
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	return windows.AssignProcessToJobObject(control.job, handle)
}

func terminateProcess(cmd *exec.Cmd, control processControl) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if control.job != 0 {
		return windows.TerminateJobObject(control.job, 1)
	}
	return cmd.Process.Kill()
}

func forceKillProcess(cmd *exec.Cmd, control processControl) error {
	return terminateProcess(cmd, control)
}

func closeProcessControl(control processControl) error {
	if control.job == 0 {
		return nil
	}
	return windows.CloseHandle(control.job)
}

func processAlreadyGone(_ error) bool {
	return false
}
