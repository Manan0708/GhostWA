package store

import (
	"testing"
	"time"
)

func TestNewStoreInMemory(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer store.Close()

	// Verify we can insert and query from our initialized schema.
	_, err = store.DB.Exec("INSERT INTO contacts (jid, name, phone) VALUES (?, ?, ?)", "12345@s.whatsapp.net", "Test Name", "12345")
	if err != nil {
		t.Fatalf("failed to insert contact: %v", err)
	}

	var name string
	err = store.DB.QueryRow("SELECT name FROM contacts WHERE jid = ?", "12345@s.whatsapp.net").Scan(&name)
	if err != nil {
		t.Fatalf("failed to query contact: %v", err)
	}

	if name != "Test Name" {
		t.Errorf("got name %q, want %q", name, "Test Name")
	}
}

func TestStoreUpsertAndSummaries(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	// Test UpsertContact
	err = s.UpsertContact("contact-jid", "Friend", "123456", "PushName")
	if err != nil {
		t.Fatalf("failed to upsert contact: %v", err)
	}

	// Test UpsertChat
	now := time.Now().Truncate(time.Second) // Truncate because SQLite DATETIME might not match sub-seconds precisely
	err = s.UpsertChat("contact-jid", "Friend", now)
	if err != nil {
		t.Fatalf("failed to upsert chat: %v", err)
	}

	// Test IncrementUnreadCount
	err = s.IncrementUnreadCount("contact-jid")
	if err != nil {
		t.Fatalf("failed to increment unread: %v", err)
	}

	// Test SaveMessage
	err = s.SaveMessage("msg-id", "contact-jid", "contact-jid", "hello test", now, false)
	if err != nil {
		t.Fatalf("failed to save message: %v", err)
	}

	// Verify GetRecentMessages
	msgs, err := s.GetRecentMessages("contact-jid", 10)
	if err != nil {
		t.Fatalf("failed to get messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Content != "hello test" {
		t.Errorf("got content %q, want %q", msgs[0].Content, "hello test")
	}

	// Verify Chat Summaries
	chats, err := s.GetChatList()
	if err != nil {
		t.Fatalf("failed to get chats: %v", err)
	}

	if len(chats) != 1 {
		t.Fatalf("got %d chats, want 1", len(chats))
	}

	c := chats[0]
	if c.JID != "contact-jid" {
		t.Errorf("got JID %q, want %q", c.JID, "contact-jid")
	}
	if c.Name != "Friend" {
		t.Errorf("got name %q, want %q", c.Name, "Friend")
	}
	if c.UnreadCount != 1 {
		t.Errorf("got unread count %d, want 1", c.UnreadCount)
	}
}

func TestResetDatabase(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	_ = s.UpsertContact("123@s.whatsapp.net", "User", "123", "User")
	_ = s.UpsertChat("123@s.whatsapp.net", "User", time.Now())
	_ = s.SaveMessage("m1", "123@s.whatsapp.net", "123@s.whatsapp.net", "test", time.Now(), false)

	err = s.ResetDatabase()
	if err != nil {
		t.Fatalf("failed to reset database: %v", err)
	}

	chats, _ := s.GetChatList()
	if len(chats) != 0 {
		t.Errorf("expected 0 chats after reset, got %d", len(chats))
	}
}
