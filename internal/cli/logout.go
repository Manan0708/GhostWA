package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	wadaemon "github.com/Manan0708/wacli/internal/daemon"
)

// runLogout handles the "logout" CLI command.
func runLogout(stdout, stderr io.Writer) int {
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	req := wadaemon.Request{Type: "logout"}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(stderr, "Error writing logout request to daemon: %v\n", err)
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
		fmt.Fprintf(stderr, "Logout failed: %s\n", resp.Error)
		return 1
	}

	fmt.Fprintln(stdout, "✓ Successfully logged out and unlinked your WhatsApp device.")
	fmt.Fprintln(stdout, "✓ Session credentials cleared.")
	return 0
}
