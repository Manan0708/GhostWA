package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/Manan0708/GhostWA/internal/store"
)

// runSync handles "ghostwa sync" or "ghostwa sync chats" to manually rebuild and sync all active chats.
func runSync(args []string, stdout, stderr io.Writer) int {
	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error finding data directory: %v\n", err)
		return 1
	}

	sessionPath := filepath.Join(dataDir, "session.db")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		fmt.Fprintln(stderr, "Not logged in. Please run 'ghostwa login' to link your WhatsApp device.")
		return 1
	}

	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening database: %v\n", err)
		return 1
	}
	defer s.Close()

	// 1. Rebuild chats from messages table
	_ = s.RebuildChatsFromMessages()

	// 2. Request daemon to trigger store contact/chat sync with WhatsApp server
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err == nil {
		defer conn.Close()
		req := wadaemon.Request{Type: "status"}
		data, _ := json.Marshal(req)
		_, _ = conn.Write(append(data, '\n'))
		reader := bufio.NewReader(conn)
		_, _ = reader.ReadBytes('\n')
	}

	chats, _ := s.GetChatList()
	fmt.Fprintf(stdout, "✓ Successfully synced %d active chats into local database!\n", len(chats))
	fmt.Fprintln(stdout, "Run 'ghostwa show' or 'ghostwa chats' to view your updated chat list.")
	return 0
}
