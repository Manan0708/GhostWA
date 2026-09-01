package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/Manan0708/GhostWA/internal/store"
)

// runListen streams and prints real-time messages forwarded by the background daemon.
func runListen(stdout, stderr io.Writer) int {
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

	// Connect to or start the daemon
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	// Subscribe to daemon messages
	req := wadaemon.Request{Type: "subscribe"}
	data, _ := json.Marshal(req)
	_, _ = conn.Write(append(data, '\n'))

	reader := bufio.NewReader(conn)
	// Read the subscription confirmation response
	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Fprintf(stderr, "Failed to subscribe to daemon events: %v\n", err)
		return 1
	}
	var resp wadaemon.Response
	_ = json.Unmarshal(line, &resp)
	if !resp.Success {
		fmt.Fprintf(stderr, "Daemon subscription failed: %s\n", resp.Error)
		return 1
	}

	fmt.Fprintln(stdout, "Listening for incoming WhatsApp messages...")
	fmt.Fprintln(stdout, "Press Ctrl+C to stop.")

	// Run reader loop in a separate goroutine
	stopLoop := make(chan struct{})
	go func() {
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				select {
				case <-stopLoop:
					return
				default:
					fmt.Fprintf(stderr, "\nConnection with daemon lost.\n")
					os.Exit(1)
				}
			}

			var evt wadaemon.Event
			_ = json.Unmarshal(line, &evt)

			if evt.Type == "message" {
				// Only display if recent
				if !evt.IsRecent {
					continue
				}

				senderName := evt.SenderName
				if senderName == "" {
					_ = s.DB.QueryRow("SELECT name FROM contacts WHERE jid = ?", evt.Sender+"@s.whatsapp.net").Scan(&senderName)
				}
				if senderName == "" {
					senderName = "+" + evt.Sender
				}

				// Check if this is a group message
				if strings.HasSuffix(evt.Chat, "@g.us") {
					groupName := ""
					_ = s.DB.QueryRow("SELECT COALESCE(name, '') FROM chats WHERE jid = ?", evt.Chat).Scan(&groupName)
					if groupName == "" {
						groupName = "Group"
					}
					fmt.Fprintf(stdout, "\n[%s] %s (in %s):\n%s\n", evt.Timestamp, senderName, groupName, evt.Body)
				} else {
					fmt.Fprintf(stdout, "\n[%s] %s:\n%s\n", evt.Timestamp, senderName, evt.Body)
				}
			}
		}
	}()

	// Block until Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	close(stopLoop)
	fmt.Fprintln(stdout, "\nStopping listener...")
	return 0
}
