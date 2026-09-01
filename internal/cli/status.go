package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
)

// runStatus queries the background daemon for the current synchronization state.
func runStatus(stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "WACLI Status")
	fmt.Fprintln(stdout, "────────────────────────")
	fmt.Fprintln(stdout)

	if os.Getenv("WACLI_TEST_MODE") == "true" {
		fmt.Fprintf(stdout, "Logged in : no\n")
		fmt.Fprintf(stdout, "Connected : no\n")
		fmt.Fprintf(stdout, "User      : —\n")
		return 0
	}

	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	req := wadaemon.Request{Type: "status"}
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

	loggedIn := "no"
	connected := "no"
	user := "—"

	if resp.Status == "connected" || resp.Status == "disconnected" {
		loggedIn = "yes"
		user = resp.Phone
		if resp.Status == "connected" {
			connected = "yes"
		}
	}

	fmt.Fprintf(stdout, "Logged in : %s\n", loggedIn)
	fmt.Fprintf(stdout, "Connected : %s\n", connected)
	fmt.Fprintf(stdout, "User      : %s\n", user)

	return 0
}
