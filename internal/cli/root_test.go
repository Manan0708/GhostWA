package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Manan0708/GhostWA/internal/store"
)

func TestRunNoArgsPrintsWACLI(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(nil, stdout, stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "GhostWA v2.5.7 — Silent, High-Performance WhatsApp Terminal Client" {
		t.Fatalf("stdout = %q, want %q", got, "GhostWA v2.5.7 — Silent, High-Performance WhatsApp Terminal Client")
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
	tempDir, err := os.MkdirTemp("", "wacli-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("WACLI_DATA_DIR", tempDir)
	t.Setenv("WACLI_TEST_MODE", "true")

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

	code := Run([]string{"foobar"}, stdout, stderr)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown command: foobar") {
		t.Fatalf("stderr = %q, want unknown-command error", stderr.String())
	}
}

func TestRunChats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wacli-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("WACLI_DATA_DIR", tempDir)
	_ = store.SaveSessionMeta("1234567890", true)
	_ = os.WriteFile(filepath.Join(tempDir, "session.db"), []byte("dummy"), 0644)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"chats"}, stdout, stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Chats") {
		t.Fatalf("chats output missing title:\n%s", out)
	}
	if !strings.Contains(out, "No saved chats found.") {
		t.Fatalf("chats output missing empty state:\n%s", out)
	}
}

func TestRunOpen(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wacli-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("WACLI_DATA_DIR", tempDir)
	t.Setenv("WACLI_TEST_MODE", "true")

	// Mock stdin to input "/exit" immediately
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r
	w.Write([]byte("/exit\n"))
	w.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	// Attempt to open an direct JID
	code := Run([]string{"open", "12345@s.whatsapp.net"}, stdout, stderr)

	// Should fail with not logged in error since the directory is empty
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "Not logged in") {
		t.Fatalf("stderr = %q, want not logged in error", stderr.String())
	}
}

func TestRunContacts(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wacli-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("WACLI_DATA_DIR", tempDir)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"contacts"}, stdout, stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Contacts List") {
		t.Fatalf("contacts output missing title:\n%s", out)
	}
	if !strings.Contains(out, "No registered contacts found.") {
		t.Fatalf("contacts output missing empty state:\n%s", out)
	}
}

func TestRunSearch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "wacli-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("WACLI_DATA_DIR", tempDir)

	dbPath := filepath.Join(tempDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Insert mock chat and messages
	_ = s.UpsertChat("123@s.whatsapp.net", "Aayushi", time.Now())
	_ = s.SaveMessage("msg1", "123@s.whatsapp.net", "123@s.whatsapp.net", "Are you going to the hackathon?", time.Now(), false)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"search", "hackathon"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}

	out := stdout.String()
	if !strings.Contains(out, "Search results for: hackathon") {
		t.Fatalf("missing query title in output: %s", out)
	}
	if !strings.Contains(out, "Aayushi") {
		t.Fatalf("missing contact name in output: %s", out)
	}
	if !strings.Contains(out, "Are you going to the hackathon?") {
		t.Fatalf("missing message content in output: %s", out)
	}
}
