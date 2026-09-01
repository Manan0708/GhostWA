package store

import (
	"time"
)

// Message represents a single message record from the SQLite database.
type Message struct {
	ID        string
	SenderJID string
	Content   string
	Timestamp time.Time
	IsFromMe  bool
	Reaction  string
}

// SaveMessage stores a message in the local SQLite table.
func (s *Store) SaveMessage(id, chatJID, senderJID, content string, timestamp time.Time, isFromMe bool) error {
	isFromMeInt := 0
	if isFromMe {
		isFromMeInt = 1
	}

	query := `
	INSERT INTO messages (id, chat_jid, sender_jid, content, timestamp, is_from_me)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO NOTHING
	`
	_, err := s.DB.Exec(query, id, chatJID, senderJID, content, timestamp, isFromMeInt)
	return err
}

// SetMessageReaction updates the reaction emoji for a given message ID.
func (s *Store) SetMessageReaction(id string, emoji string) error {
	_, err := s.DB.Exec("UPDATE messages SET reaction = ? WHERE id = ?", emoji, id)
	return err
}

// GetLastMessage retrieves the most recent message in a chat.
func (s *Store) GetLastMessage(chatJID string) (*Message, error) {
	query := `
	SELECT id, sender_jid, content, timestamp, is_from_me, COALESCE(reaction, '')
	FROM messages
	WHERE chat_jid = ?
	ORDER BY timestamp DESC
	LIMIT 1
	`
	var m Message
	var isFromMeInt int
	err := s.DB.QueryRow(query, chatJID).Scan(&m.ID, &m.SenderJID, &m.Content, &m.Timestamp, &isFromMeInt, &m.Reaction)
	if err != nil {
		return nil, err
	}
	m.IsFromMe = (isFromMeInt == 1)
	return &m, nil
}

// GetRecentMessages retrieves the latest messages for a chat, sorted in chronological order.
func (s *Store) GetRecentMessages(chatJID string, limit int) ([]Message, error) {
	query := `
	SELECT id, sender_jid, content, timestamp, is_from_me, COALESCE(reaction, '')
	FROM messages
	WHERE chat_jid = ?
	ORDER BY timestamp DESC
	LIMIT ?
	`
	rows, err := s.DB.Query(query, chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		var isFromMeInt int
		err := rows.Scan(&m.ID, &m.SenderJID, &m.Content, &m.Timestamp, &isFromMeInt, &m.Reaction)
		if err != nil {
			return nil, err
		}
		m.IsFromMe = (isFromMeInt == 1)
		msgs = append(msgs, m)
	}

	// Reverse the slice so the messages are returned in chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}

// MessageSearchResult contains a message match along with chat info.
type MessageSearchResult struct {
	ID        string
	SenderJID string
	Content   string
	Timestamp time.Time
	IsFromMe  bool
	ChatJID   string
	ChatName  string
}

// SearchMessages queries the local SQLite database for messages containing the query string.
func (s *Store) SearchMessages(query string) ([]MessageSearchResult, error) {
	sqlQuery := `
	SELECT m.id, m.chat_jid, m.sender_jid, m.content, m.timestamp, m.is_from_me, COALESCE(c.name, c.jid)
	FROM messages m
	LEFT JOIN chats c ON m.chat_jid = c.jid
	WHERE m.content LIKE ?
	ORDER BY m.timestamp DESC
	`
	rows, err := s.DB.Query(sqlQuery, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MessageSearchResult
	for rows.Next() {
		var r MessageSearchResult
		var isFromMeInt int
		err := rows.Scan(&r.ID, &r.ChatJID, &r.SenderJID, &r.Content, &r.Timestamp, &isFromMeInt, &r.ChatName)
		if err != nil {
			return nil, err
		}
		r.IsFromMe = (isFromMeInt == 1)
		results = append(results, r)
	}
	return results, nil
}

