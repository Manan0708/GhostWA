package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
)

// runDeleteChat handles the ghostwa delete-chat <recipient> command.
func runDeleteChat(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Error: missing recipient name or phone number.")
		fmt.Fprintln(stderr, "Usage: ghostwa delete-chat <recipient>")
		return 1
	}

	recipient := args[0]

	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	req := wadaemon.Request{
		Type: "delete_chat",
		To:   recipient,
	}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(stderr, "Error sending request to daemon: %v\n", err)
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
		fmt.Fprintf(stderr, "Error deleting chat: %s\n", resp.Error)
		return 1
	}

	fmt.Fprintf(stdout, "✓ Chat with '%s' and all its message logs have been deleted successfully.\n", recipient)
	return 0
}
