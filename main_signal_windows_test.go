//go:build windows

package main

import "testing"

func terminateCurrentProcess(t *testing.T) {
	t.Helper()
	// The caller skips the Unix signal integration test on Windows.
}
