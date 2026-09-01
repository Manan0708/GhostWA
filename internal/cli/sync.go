package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
)

// runSync handles ghostwa sync <chats|history> [recipient] commands.
func runSync(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Error: missing sync target.")
		fmt.Fprintln(stderr, "Usage: ghostwa sync <chats|history> [recipient]")
		return 1
	}

	target := strings.ToLower(args[0])
	recipient := ""
	if len(args) > 1 {
		recipient = args[1]
	}

	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	reqType := "sync_chats"
	if target == "history" {
		reqType = "sync_history"
	}

	req := wadaemon.Request{
		Type: reqType,
		To:   recipient,
	}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(stderr, "Error sending sync instruction to daemon: %v\n", err)
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
		fmt.Fprintf(stderr, "Sync error from daemon: %s\n", resp.Error)
		return 1
	}

	if target == "history" {
		if recipient != "" {
			fmt.Fprintf(stdout, "✓ History refresh initiated for '%s'.\n", recipient)
		} else {
			fmt.Fprintln(stdout, "✓ Full history synchronization and refresh initiated.")
		}
	} else {
		fmt.Fprintln(stdout, "✓ Contacts and group chat names re-synchronized successfully.")
	}

	return 0
}
