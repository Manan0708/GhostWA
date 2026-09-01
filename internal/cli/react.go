package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
)

// runReact handles "ghostwa react <chat> <emoji>" or "ghostwa react <chat> <msg_id> <emoji>"
func runReact(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "Usage: ghostwa react <chat_or_phone> <emoji>")
		fmt.Fprintln(stderr, "   or: ghostwa react <chat_or_phone> <msg_id> <emoji>")
		return 1
	}

	target := args[0]
	emoji := args[len(args)-1]
	msgID := ""
	if len(args) >= 3 {
		msgID = args[1]
	}

	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	req := wadaemon.Request{
		Type:  "react",
		To:    target,
		MsgID: msgID,
		Emoji: emoji,
	}
	data, _ := json.Marshal(req)
	_, _ = conn.Write(append(data, '\n'))

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Fprintf(stderr, "Connection lost with daemon: %v\n", err)
		return 1
	}

	var resp wadaemon.Response
	_ = json.Unmarshal(line, &resp)
	if !resp.Success {
		fmt.Fprintf(stderr, "Error sending reaction: %s\n", resp.Error)
		return 1
	}

	fmt.Fprintf(stdout, "✓ Reaction %s sent successfully to %s!\n", emoji, target)
	return 0
}
