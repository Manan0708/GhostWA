package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Manan0708/wacli/internal/store"
)

// runChats handles the "chats" command to show direct chats (excluding groups).
func runChats(stdout, stderr io.Writer) int {
	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error finding data directory: %v\n", err)
		return 1
	}

	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening database: %v\n", err)
		return 1
	}
	defer s.Close()

	chatList, err := s.GetChatList()
	if err != nil {
		fmt.Fprintf(stderr, "Error querying chat list: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Chats")
	fmt.Fprintln(stdout, "────────────────────────────────────")
	fmt.Fprintln(stdout)

	count := 0
	for _, c := range chatList {
		if strings.HasSuffix(c.JID, "@g.us") {
			continue
		}
		count++
		unreadSuffix := ""
		if c.UnreadCount > 0 {
			unreadSuffix = fmt.Sprintf("   [%d unread]", c.UnreadCount)
		}
		fmt.Fprintf(stdout, "  %-25s%s\n", c.Name, unreadSuffix)
	}

	if count == 0 {
		fmt.Fprintln(stdout, "  No active chats found.")
	}

	return 0
}

// runGroups handles the "groups" command to show only group chats.
func runGroups(stdout, stderr io.Writer) int {
	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error finding data directory: %v\n", err)
		return 1
	}

	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening database: %v\n", err)
		return 1
	}
	defer s.Close()

	chatList, err := s.GetChatList()
	if err != nil {
		fmt.Fprintf(stderr, "Error querying chat list: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Group Chats")
	fmt.Fprintln(stdout, "────────────────────────────────────")
	fmt.Fprintln(stdout)

	count := 0
	for _, c := range chatList {
		if !strings.HasSuffix(c.JID, "@g.us") {
			continue
		}
		count++
		unreadSuffix := ""
		if c.UnreadCount > 0 {
			unreadSuffix = fmt.Sprintf("   [%d unread]", c.UnreadCount)
		}
		fmt.Fprintf(stdout, "  %-25s%s\n", c.Name, unreadSuffix)
	}

	if count == 0 {
		fmt.Fprintln(stdout, "  No active group chats found.")
	}

	return 0
}
