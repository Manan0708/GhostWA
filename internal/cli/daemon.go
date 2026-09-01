package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/Manan0708/GhostWA/internal/store"
)

// runDaemonRun runs the daemon server in the current process (called by the background detached process)
func runDaemonRun(stdout, stderr io.Writer) int {
	dataDir, err := store.GetDefaultDataDir()
	if err != nil {
		fmt.Fprintf(stderr, "Error resolving default data directory: %v\n", err)
		return 1
	}

	// Set up log file redirection for the background process
	logPath := filepath.Join(dataDir, "daemon.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		log.Println("--- Daemon starting ---")
	}

	srv, err := wadaemon.NewServer(dataDir)
	if err != nil {
		log.Printf("Failed to initialize daemon server: %v", err)
		return 1
	}

	err = srv.Run()
	if err != nil {
		log.Printf("Daemon server execution error: %v", err)
		return 1
	}

	return 0
}

// runDaemonCLI handles the user management commands: wacli daemon <start|stop|status|restart>
func runDaemonCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Usage: wacli daemon <start|stop|status|restart>")
		return 1
	}

	cmd := strings.ToLower(args[0])
	switch cmd {
	case "start":
		conn, err := net.DialTimeout("tcp", "127.0.0.1:9090", 200*time.Millisecond)
		if err == nil {
			conn.Close()
			fmt.Fprintln(stdout, "Daemon is already running.")
			return 0
		}
		fmt.Fprintln(stdout, "Starting daemon in background...")
		err = wadaemon.StartDaemonProcess()
		if err != nil {
			fmt.Fprintf(stderr, "Error starting daemon: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "✓ Daemon launched successfully.")

	case "stop":
		fmt.Fprintln(stdout, "Stopping background daemon...")
		err := wadaemon.StopDaemon()
		if err != nil {
			fmt.Fprintf(stderr, "Error stopping daemon (or daemon is not running): %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "✓ Daemon stopped successfully.")

	case "status":
		conn, err := net.DialTimeout("tcp", "127.0.0.1:9090", 200*time.Millisecond)
		if err != nil {
			fmt.Fprintln(stdout, "Daemon status: OFFLINE")
			return 0
		}
		defer conn.Close()

		// Request connection and login status from daemon
		req := wadaemon.Request{Type: "status"}
		data, _ := json.Marshal(req)
		_, _ = conn.Write(append(data, '\n'))

		// Read response
		reader := bufio.NewReader(conn)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Fprintln(stdout, "Daemon status: ONLINE (Unresponsive)")
			return 0
		}
		var resp wadaemon.Response
		_ = json.Unmarshal(line, &resp)

		fmt.Fprintln(stdout, "Daemon status: ONLINE")
		fmt.Fprintf(stdout, "WhatsApp Connection: %s\n", resp.Status)
		fmt.Fprintf(stdout, "User Profile:        %s\n", resp.Phone)

	case "restart":
		_ = wadaemon.StopDaemon()
		time.Sleep(500 * time.Millisecond)
		fmt.Fprintln(stdout, "Starting daemon...")
		err := wadaemon.StartDaemonProcess()
		if err != nil {
			fmt.Fprintf(stderr, "Error starting daemon: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "✓ Daemon restarted.")

	default:
		fmt.Fprintf(stderr, "Unknown daemon command: %s\n", cmd)
		fmt.Fprintln(stderr, "Usage: wacli daemon <start|stop|status|restart>")
		return 1
	}

	return 0
}
