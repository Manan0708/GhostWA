package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ChatSummary represents a single active conversation metadata.
type ChatSummary struct {
	JID             string
	Name            string
	UnreadCount     int
	LastMessageTime time.Time
}

// UpsertChat registers a chat JID or updates its metadata.
func (s *Store) UpsertChat(jid, name string, lastMessageTime time.Time) error {
	if lastMessageTime.IsZero() {
		lastMessageTime = time.Now()
	}

	// Query to preserve existing chat names if we receive a message without contact cache resolved.
	var existingName sql.NullString
	err := s.DB.QueryRow("SELECT name FROM chats WHERE jid = ?", jid).Scan(&existingName)
	if err == nil && existingName.Valid && name == "" {
		name = existingName.String
	}

	query := `
	INSERT INTO chats (jid, name, last_message_time, updated_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = COALESCE(NULLIF(excluded.name, ''), chats.name),
		last_message_time = excluded.last_message_time,
		updated_at = excluded.updated_at
	`
	_, err = s.DB.Exec(query, jid, name, lastMessageTime, time.Now())
	return err
}

// IncrementUnreadCount increments unread counter by 1.
func (s *Store) IncrementUnreadCount(jid string) error {
	_, err := s.DB.Exec("UPDATE chats SET unread_count = unread_count + 1 WHERE jid = ?", jid)
	return err
}

// ResetUnreadCount sets unread count to 0.
func (s *Store) ResetUnreadCount(jid string) error {
	_, err := s.DB.Exec("UPDATE chats SET unread_count = 0 WHERE jid = ?", jid)
	return err
}

// SetUnreadCount sets unread count to an explicit value.
func (s *Store) SetUnreadCount(jid string, count int) error {
	_, err := s.DB.Exec("UPDATE chats SET unread_count = ? WHERE jid = ?", count, jid)
	return err
}

// GetChatList retrieves all registered chats sorted by active timestamp.
func (s *Store) GetChatList() ([]ChatSummary, error) {
	query := `
	SELECT 
		c.jid, 
		COALESCE(NULLIF(c.name, ''), NULLIF(con.name, ''), NULLIF(con.push_name, ''), c.jid) as resolved_name, 
		c.unread_count, 
		c.last_message_time
	FROM chats c
	LEFT JOIN contacts con ON c.jid = con.jid
	ORDER BY c.last_message_time DESC
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []ChatSummary
	for rows.Next() {
		var c ChatSummary
		err := rows.Scan(&c.JID, &c.Name, &c.UnreadCount, &c.LastMessageTime)
		if err != nil {
			return nil, err
		}
		c.Name = formatDisplayName(c.JID, c.Name)
		chats = append(chats, c)
	}
	return chats, nil
}

// formatDisplayName formats a JID or raw phone number into a clean human-readable name or phone string.
func formatDisplayName(jid, name string) string {
	if name != "" && name != jid && !strings.Contains(name, "@s.whatsapp.net") && !strings.Contains(name, "@g.us") {
		return name
	}
	if strings.HasSuffix(jid, "@s.whatsapp.net") {
		num := strings.TrimSuffix(jid, "@s.whatsapp.net")
		if len(num) >= 10 {
			if len(num) == 12 && strings.HasPrefix(num, "91") {
				return fmt.Sprintf("+91 %s-%s", num[2:7], num[7:])
			} else if len(num) == 11 && strings.HasPrefix(num, "1") {
				return fmt.Sprintf("+1 (%s) %s-%s", num[1:4], num[4:7], num[7:])
			}
			return "+" + num
		}
		return "+" + num
	} else if strings.HasSuffix(jid, "@g.us") {
		return "Group Chat"
	}
	return jid
}

// DeleteChat purges a chat from SQLite storage along with all its message logs.
func (s *Store) DeleteChat(jid string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, _ = tx.Exec("DELETE FROM messages WHERE chat_jid = ?", jid)
	_, _ = tx.Exec("DELETE FROM chats WHERE jid = ?", jid)

	return tx.Commit()
}
