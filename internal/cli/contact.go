package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/Manan0708/wacli/internal/store"
)

// runAddContact handles the "add-contact" command: wacli add-contact <name> <phone_number>
func runAddContact(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "Error: missing contact name or phone number.")
		fmt.Fprintln(stderr, "Usage: wacli add-contact <name> <phone_number>")
		return 1
	}

	name := strings.TrimSpace(args[0])
	phone := strings.TrimSpace(args[1])

	// Clean formatting symbols
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// Basic digits validation
	isNumeric := true
	for _, c := range phone {
		if c < '0' || c > '9' {
			isNumeric = false
			break
		}
	}
	if !isNumeric || len(phone) < 8 {
		fmt.Fprintln(stderr, "Error: phone number must contain only digits (including country code).")
		return 1
	}

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

	jid := phone + "@s.whatsapp.net"
	
	// Save the contact details
	err = s.UpsertContact(jid, name, phone, "")
	if err != nil {
		fmt.Fprintf(stderr, "Error saving contact to database: %v\n", err)
		return 1
	}

	// Upsert the chat metadata so it appears on the chat list
	err = s.UpsertChat(jid, name, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "Error updating chat list: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "✓ Contact '%s' successfully linked to +%s\n", name, phone)
	return 0
}

// runContacts list all registered contacts in the SQLite database.
func runContacts(stdout, stderr io.Writer) int {
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

	rows, err := s.DB.Query("SELECT jid, name, phone, push_name FROM contacts ORDER BY name ASC")
	if err != nil {
		fmt.Fprintf(stderr, "Error querying database contacts: %v\n", err)
		return 1
	}
	defer rows.Close()

	fmt.Fprintln(stdout, "Contacts List")
	fmt.Fprintln(stdout, "────────────────────────────────────")
	fmt.Fprintln(stdout)

	count := 0
	for rows.Next() {
		var jid, name, phone, pushName string
		err := rows.Scan(&jid, &name, &phone, &pushName)
		if err != nil {
			fmt.Fprintf(stderr, "Error reading contact row: %v\n", err)
			return 1
		}
		displayName := name
		if displayName == "" {
			displayName = pushName
		}
		if displayName == "" {
			displayName = "[No Name]"
		}
		fmt.Fprintf(stdout, "  %-15s : +%s (%s)\n", displayName, phone, jid)
		count++
	}

	if count == 0 {
		fmt.Fprintln(stdout, "  No registered contacts found.")
	}

	return 0
}
