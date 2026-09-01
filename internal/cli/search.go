package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Manan0708/GhostWA/internal/store"
)

// runSearch searches the local SQLite database for matching contact/chat names and message contents.
func runSearch(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Error: Missing search query.")
		fmt.Fprintln(stderr, "Usage: wacli search <text>")
		return 1
	}

	query := strings.Join(args, " ")

	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error getting data directory: %v\n", err)
		return 1
	}

	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening database: %v\n", err)
		return 1
	}
	defer s.Close()

	// 1. Search for matching contacts/chats by name (case-insensitive and partial match)
	var matchedChats []struct{ JID, Name string }
	rows, err := s.DB.Query(`
		SELECT DISTINCT jid, name FROM chats WHERE name LIKE ? COLLATE NOCASE
		UNION
		SELECT DISTINCT jid, name FROM contacts WHERE name LIKE ? COLLATE NOCASE
	`, "%"+query+"%", "%"+query+"%")
	if err == nil {
		for rows.Next() {
			var r struct{ JID, Name string }
			if err := rows.Scan(&r.JID, &r.Name); err == nil && r.Name != "" {
				matchedChats = append(matchedChats, r)
			}
		}
		rows.Close()
	}

	fmt.Fprintf(stdout, "Search results for: %s\n", query)
	fmt.Fprintln(stdout, strings.Repeat("─", 40))
	fmt.Fprintln(stdout)

	if len(matchedChats) > 0 {
		fmt.Fprintln(stdout, "Matching Chats & Contacts:")
		for _, mc := range matchedChats {
			typeStr := "Direct Chat"
			if strings.HasSuffix(mc.JID, "@g.us") {
				typeStr = "Group Chat"
			}
			fmt.Fprintf(stdout, "  • %s (%s)\n", mc.Name, typeStr)
		}
		fmt.Fprintln(stdout)
	}

	// 2. Search for matching messages
	results, err := s.SearchMessages(query)
	if err != nil {
		fmt.Fprintf(stderr, "Error performing search query: %v\n", err)
		return 1
	}

	if len(results) > 0 {
		fmt.Fprintln(stdout, "Matching Messages:")
		for _, res := range results {
			displayName := res.ChatName
			if strings.HasSuffix(res.ChatJID, "@g.us") {
				senderName := res.SenderJID
				if res.IsFromMe {
					senderName = "You"
				} else {
					var name string
					err := s.DB.QueryRow("SELECT COALESCE(name, push_name, jid) FROM contacts WHERE jid = ?", res.SenderJID).Scan(&name)
					if err == nil && name != "" {
						senderName = name
					}
				}
				displayName = fmt.Sprintf("%s (%s)", res.ChatName, senderName)
			} else if res.IsFromMe {
				displayName = fmt.Sprintf("%s (You)", res.ChatName)
			}

			timeStr := res.Timestamp.Local().Format("Jan 02, 15:04")
			fmt.Fprintf(stdout, "  [%s] %s: \"%s\"\n", timeStr, displayName, res.Content)
		}
		fmt.Fprintln(stdout)
	}

	if len(matchedChats) == 0 && len(results) == 0 {
		fmt.Fprintln(stdout, "No matching contacts, chats, or messages found.")
	}

	return 0
}
