package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunNoArgsPrintsWACLI(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(nil, stdout, stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "WACLI" {
		t.Fatalf("stdout = %q, want %q", got, "WACLI")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"--help"}, stdout, stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("help output missing Usage:\n%s", out)
	}
	if !strings.Contains(out, "status") {
		t.Fatalf("help output missing status command:\n%s", out)
	}
}

func TestRunStatus(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"status"}, stdout, stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "WACLI Status") {
		t.Fatalf("status output missing title:\n%s", out)
	}
	if !strings.Contains(out, "Logged in : no") {
		t.Fatalf("status output missing logged-in line:\n%s", out)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"send"}, stdout, stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown command: send") {
		t.Fatalf("stderr = %q, want unknown-command error", stderr.String())
	}
}
