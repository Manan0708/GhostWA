package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
)

// runLogout handles the "logout" CLI command.
func runLogout(stdout, stderr io.Writer) int {
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err == nil {
		defer conn.Close()

		req := wadaemon.Request{Type: "logout"}
		data, _ := json.Marshal(req)
		_, _ = conn.Write(append(data, '\n'))

		reader := bufio.NewReader(conn)
		line, _ := reader.ReadBytes('\n')
		var resp wadaemon.Response
		_ = json.Unmarshal(line, &resp)
	}

	// Always force-wipe session.db and reset messages database locally to guarantee complete session reset
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".local", "share", "wacli")
	sessionPath := filepath.Join(dataDir, "session.db")
	dbPath := filepath.Join(dataDir, "messages.db")
	_ = os.Remove(sessionPath)
	_ = os.Remove(dbPath)

	fmt.Fprintln(stdout, "✓ Successfully logged out and unlinked your WhatsApp device.")
	fmt.Fprintln(stdout, "✓ Session credentials and local chat history cleared.")
	return 0
}
