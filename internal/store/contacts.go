package store

import (
	"time"
)

// UpsertContact inserts a contact or updates their details on collision.
func (s *Store) UpsertContact(jid, name, phone, pushName string) error {
	query := `
	INSERT INTO contacts (jid, name, phone, push_name, updated_at)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = excluded.name,
		phone = excluded.phone,
		push_name = COALESCE(excluded.push_name, contacts.push_name),
		updated_at = excluded.updated_at
	`
	_, err := s.DB.Exec(query, jid, name, phone, pushName, time.Now())
	return err
}
