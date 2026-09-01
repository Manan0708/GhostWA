package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/mdp/qrterminal/v3"
)

// runLogin handles the "login" command by talking to the background daemon.
func runLogin(stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "WhatsApp Login")
	fmt.Fprintln(stdout, "────────────────────────")
	fmt.Fprintln(stdout)

	// Connect to or start the daemon process
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	// Send login request
	req := wadaemon.Request{Type: "login"}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(stderr, "Error writing login request to daemon: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Scan this QR code using:")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "WhatsApp")
	fmt.Fprintln(stdout, "→ Settings")
	fmt.Fprintln(stdout, "→ Linked Devices")
	fmt.Fprintln(stdout, "→ Link a Device")
	fmt.Fprintln(stdout)

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Fprintf(stderr, "Connection lost with daemon: %v\n", err)
			return 1
		}

		var evt wadaemon.Event
		if err := json.Unmarshal(line, &evt); err == nil && evt.Type != "" {
			switch evt.Type {
			case "qr":
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, stdout)
				fmt.Fprintln(stdout)
				fmt.Fprintln(stdout, "Waiting for authentication...")
			case "login_success":
				fmt.Fprintln(stdout)
				fmt.Fprintln(stdout, "✓ Successfully linked")
				fmt.Fprintln(stdout, "✓ Session saved")
				return 0
			case "error":
				if evt.Code == "timeout" {
					fmt.Fprintf(stderr, "Pairing error: pairing timeout reached.\n")
				} else {
					fmt.Fprintf(stderr, "Pairing error encountered.\n")
				}
				return 1
			}
		} else {
			var resp wadaemon.Response
			_ = json.Unmarshal(line, &resp)
			if !resp.Success {
				fmt.Fprintf(stderr, "Login error: %s\n", resp.Error)
				return 1
			}
		}
	}
}
