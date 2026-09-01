package whatsapp

import (
	"os"
	"testing"

	"github.com/Manan0708/GhostWA/internal/store"
)

func TestNewClient(t *testing.T) {
	// Create a temporary directory for the session database.
	tempDir, err := os.MkdirTemp("", "wacli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	client, err := NewClient(tempDir, s)
	if err != nil {
		t.Fatalf("failed to initialize whatsmeow client: %v", err)
	}
	defer client.Close()

	if client.IsLoggedIn() {
		t.Error("expected new client to not be logged in")
	}

	if client.IsConnected() {
		t.Error("expected new client to not be connected")
	}
}
