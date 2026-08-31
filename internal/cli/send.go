package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	wadaemon "github.com/Manan0708/wacli/internal/daemon"
	"github.com/Manan0708/wacli/internal/resolver"
	"github.com/Manan0708/wacli/internal/store"
)

// runSend sends a text message to a recipient by instructing the background daemon.
func runSend(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "Error: missing arguments.")
		fmt.Fprintln(stderr, "Usage: wacli send <recipient> <message>")
		return 1
	}

	recipient := args[0]
	messageText := args[1]

	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error finding data directory: %v\n", err)
		return 1
	}

	// Open the SQLite database to resolve contacts locally
	dbPath := filepath.Join(dataDir, "messages.db")
	s, err := store.NewStore(dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error opening database: %v\n", err)
		return 1
	}
	defer s.Close()

	// Resolve the JID
	res := resolver.NewResolver(s)
	targetJID, err := res.Resolve(recipient)
	if err != nil {
		fmt.Fprintf(stderr, "Resolution error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "→ Resolved '%s' to JID: %s\n", recipient, targetJID.String())

	// Connect to or start the daemon
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	// Send message send instruction to daemon
	req := wadaemon.Request{
		Type: "send",
		To:   targetJID.String(),
		Body: messageText,
	}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(stderr, "Error sending instruction to daemon: %v\n", err)
		return 1
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Fprintf(stderr, "Connection lost with daemon: %v\n", err)
		return 1
	}

	var resp wadaemon.Response
	_ = json.Unmarshal(line, &resp)

	if !resp.Success {
		fmt.Fprintf(stderr, "Error from daemon: %s\n", resp.Error)
		return 1
	}

	fmt.Fprintf(stdout, "✓ Message sent successfully via daemon (ID: %s)\n", resp.MsgID)
	return 0
}
