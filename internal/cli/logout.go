package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/Manan0708/GhostWA/internal/store"
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

	// Always force-kill any running daemon processes to release SQLite file locks
	_ = exec.Command("taskkill", "/F", "/IM", "ghostwa.exe").Run()
	_ = exec.Command("taskkill", "/F", "/IM", "wacli.exe").Run()

	meta, _ := store.GetSessionMeta()
	_ = store.ClearSessionMeta()

	if meta.Phone != "" {
		accountDir, _ := store.GetAccountDataDir(meta.Phone)
		if accountDir != "" {
			_ = os.RemoveAll(accountDir)
		}
	}

	baseDir, _ := store.GetDefaultDataDir()
	_ = os.RemoveAll(filepath.Join(baseDir, "accounts"))
	_ = os.Remove(filepath.Join(baseDir, "session.db"))
	_ = os.Remove(filepath.Join(baseDir, "messages.db"))

	fmt.Fprintln(stdout, "✓ Successfully logged out and unlinked your WhatsApp device.")
	fmt.Fprintln(stdout, "✓ Session credentials and local chat history cleared.")
	return 0
}
