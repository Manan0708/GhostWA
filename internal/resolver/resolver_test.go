package resolver

import (
	"testing"

	"github.com/Manan0708/GhostWA/internal/store"
)

func TestResolveDirectJID(t *testing.T) {
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	r := NewResolver(s)

	// Direct JID parsing
	jid, err := r.Resolve("123456@s.whatsapp.net")
	if err != nil {
		t.Fatalf("unexpected error resolving JID: %v", err)
	}
	if jid.String() != "123456@s.whatsapp.net" {
		t.Errorf("got %q, want %q", jid.String(), "123456@s.whatsapp.net")
	}
}

func TestResolvePhoneNumber(t *testing.T) {
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	r := NewResolver(s)

	// Phone number parsing with + prefix
	jid, err := r.Resolve("+911234567890")
	if err != nil {
		t.Fatalf("unexpected error resolving phone number: %v", err)
	}
	if jid.String() != "911234567890@s.whatsapp.net" {
		t.Errorf("got %q, want %q", jid.String(), "911234567890@s.whatsapp.net")
	}
}

func TestResolveContactName(t *testing.T) {
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer s.Close()

	// Insert a mock contact into the test database
	_, err = s.DB.Exec("INSERT INTO contacts (jid, name, phone) VALUES (?, ?, ?)", "98765@s.whatsapp.net", "Aayushi", "98765")
	if err != nil {
		t.Fatalf("failed to insert mock contact: %v", err)
	}

	r := NewResolver(s)

	// Contact name resolution
	jid, err := r.Resolve("Aayushi")
	if err != nil {
		t.Fatalf("unexpected error resolving name: %v", err)
	}
	if jid.String() != "98765@s.whatsapp.net" {
		t.Errorf("got %q, want %q", jid.String(), "98765@s.whatsapp.net")
	}

	// Case-insensitive resolution check
	jid, err = r.Resolve("aayushi")
	if err != nil {
		t.Fatalf("unexpected error resolving lowercase name: %v", err)
	}
	if jid.String() != "98765@s.whatsapp.net" {
		t.Errorf("got %q, want %q", jid.String(), "98765@s.whatsapp.net")
	}
}
