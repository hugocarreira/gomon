//go:build !windows

package main

import (
	"os"
	"syscall"
	"testing"
)

func terminateCurrentProcess(t *testing.T) {
	t.Helper()
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send termination signal: %v", err)
	}
}
