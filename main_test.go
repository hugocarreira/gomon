package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected version command to succeed, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "gomon version") {
		t.Fatalf("unexpected version output: %q", stdout.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected help command to succeed, got %d: %s", code, stderr.String())
	}
	for _, expected := range []string{"-config", "-log-level", "project_path"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("help output missing %q: %s", expected, stdout.String())
		}
	}
}

func TestRunRejectsAmbiguousProjectPaths(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-path", "/tmp/one", "/tmp/two"}, &stdout, &stderr); code != 2 {
		t.Fatalf("expected ambiguous path to return 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "either --path") {
		t.Fatalf("unexpected error: %s", stderr.String())
	}
}

func TestRunUsesPositionalArgumentAfterFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-debounce", "1", "/definitely/missing-project"}, &stdout, &stderr); code != 1 {
		t.Fatalf("expected missing project to return 1, got %d", code)
	}
	if strings.Contains(stderr.String(), "-debounce") {
		t.Fatalf("flag name was incorrectly treated as project path: %s", stderr.String())
	}
}
