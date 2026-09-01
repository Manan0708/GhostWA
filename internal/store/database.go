package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Store wraps the sql.DB object to provide methods for querying our local SQLite database.
type Store struct {
	DB *sql.DB
}

// SessionMeta tracks active login state and associated phone number.
type SessionMeta struct {
	LoggedIn bool   `json:"logged_in"`
	Phone    string `json:"phone"`
}

// GetDefaultDataDir returns the path to ~/.local/share/wacli on Unix/macOS or the equivalent on Windows.
// It can be overridden for testing by setting the WACLI_DATA_DIR environment variable.
func GetDefaultDataDir() (string, error) {
	if envPath := os.Getenv("WACLI_DATA_DIR"); envPath != "" {
		return envPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	dataDir := filepath.Join(home, ".local", "share", "wacli")
	return dataDir, nil
}

// GetSessionMetaPath returns the path to ~/.local/share/wacli/session_info.json
func GetSessionMetaPath() (string, error) {
	baseDir, err := GetDefaultDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "session_info.json"), nil
}

// SaveSessionMeta records active login status and phone number.
func SaveSessionMeta(phone string, loggedIn bool) error {
	metaPath, err := GetSessionMetaPath()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(metaPath), 0755)
	meta := SessionMeta{
		LoggedIn: loggedIn,
		Phone:    phone,
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, data, 0644)
}

// GetSessionMeta reads active login status and phone number.
func GetSessionMeta() (SessionMeta, error) {
	metaPath, err := GetSessionMetaPath()
	if err != nil {
		return SessionMeta{LoggedIn: false}, nil
	}
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return SessionMeta{LoggedIn: false}, nil
	}
	var meta SessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return SessionMeta{LoggedIn: false}, nil
	}
	return meta, nil
}

// ClearSessionMeta wipes active session metadata.
func ClearSessionMeta() error {
	metaPath, err := GetSessionMetaPath()
	if err != nil {
		return err
	}
	_ = os.Remove(metaPath)
	return nil
}

// GetAccountDataDir returns isolated data directory for specific phone number.
func GetAccountDataDir(phone string) (string, error) {
	baseDir, err := GetDefaultDataDir()
	if err != nil {
		return "", err
	}
	cleanPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	if cleanPhone == "" {
		cleanPhone = "default"
	}
	return filepath.Join(baseDir, "accounts", cleanPhone), nil
}

// GetActiveAccountDir returns isolated account directory for currently active session.
func GetActiveAccountDir() (string, error) {
	meta, err := GetSessionMeta()
	if err != nil || !meta.LoggedIn || meta.Phone == "" {
		return "", fmt.Errorf("no active logged in session")
	}
	return GetAccountDataDir(meta.Phone)
}

// NewStore initializes a SQLite store at the specified path and runs initial table creation.
func NewStore(dbPath string) (*Store, error) {
	// If not in-memory, create directory if not exists
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for database: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign key constraints (SQLite disables them by default)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL journal mode for concurrent reads & writes
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL journal mode: %w", err)
	}

	s := &Store{DB: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.DB.Close()
}

// initSchema creates the database tables if they do not already exist.
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS contacts (
		jid TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		phone TEXT NOT NULL,
		push_name TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS chats (
		jid TEXT PRIMARY KEY,
		name TEXT,
		unread_count INTEGER DEFAULT 0,
		last_message_time DATETIME,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		chat_jid TEXT,
		sender_jid TEXT,
		content TEXT,
		timestamp DATETIME,
		is_from_me INTEGER,
		reaction TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(chat_jid) REFERENCES chats(jid)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_content ON messages(content);
	`
	if _, err := s.DB.Exec(schema); err != nil {
		return err
	}

	// Safely add reaction column if upgrading from earlier database version
	_, _ = s.DB.Exec("ALTER TABLE messages ADD COLUMN reaction TEXT DEFAULT '';")
	return nil
}

// ResetDatabase truncates all chats, messages, and contacts tables cleanly.
func (s *Store) ResetDatabase() error {
	queries := []string{
		"DELETE FROM messages;",
		"DELETE FROM chats;",
		"DELETE FROM contacts;",
		"VACUUM;",
	}
	for _, q := range queries {
		if _, err := s.DB.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
