package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Manan0708/GhostWA/internal/store"
)

func isSavedContact(c store.ChatSummary) bool {
	phoneDigits := strings.TrimSuffix(c.JID, "@s.whatsapp.net")
	cleanName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(c.Name, "+", ""), " ", ""), "-", ""), "(", "")
	cleanName = strings.ReplaceAll(cleanName, ")", "")
	return cleanName != phoneDigits && c.Name != c.JID
}

// runChats handles the "chats" command with options for saved, unsaved, and all direct chats.
func runChats(args []string, stdout, stderr io.Writer) int {
	filterMode := "saved" // default: show only saved contacts
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "unsaved", "--unsaved", "-u":
			filterMode = "unsaved"
		case "all", "--all", "-a":
			filterMode = "all"
		case "saved", "--saved", "-s":
			filterMode = "saved"
		}
	}

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

	chatList, err := s.GetChatList()
	if err != nil {
		fmt.Fprintf(stderr, "Error querying chat list: %v\n", err)
		return 1
	}

	headerTitle := "Saved Chats"
	if filterMode == "unsaved" {
		headerTitle = "Unsaved Number Chats"
	} else if filterMode == "all" {
		headerTitle = "All Direct Chats (Saved & Unsaved)"
	}

	fmt.Fprintln(stdout, headerTitle)
	fmt.Fprintln(stdout, "──────────────────────────────────────────────────")
	fmt.Fprintln(stdout)

	count := 0
	for _, c := range chatList {
		if strings.HasSuffix(c.JID, "@g.us") {
			continue
		}

		saved := isSavedContact(c)
		if filterMode == "saved" && !saved {
			continue
		} else if filterMode == "unsaved" && saved {
			continue
		}

		count++
		unreadSuffix := ""
		if c.UnreadCount > 0 {
			unreadSuffix = fmt.Sprintf("   [%d unread]", c.UnreadCount)
		}
		fmt.Fprintf(stdout, "  %-30s%s\n", c.Name, unreadSuffix)
	}

	if count == 0 {
		fmt.Fprintf(stdout, "  No %s found.\n", strings.ToLower(headerTitle))
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
