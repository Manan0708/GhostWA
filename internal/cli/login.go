package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	wadaemon "github.com/Manan0708/GhostWA/internal/daemon"
	"github.com/mdp/qrterminal/v3"
)

// runLogin handles the "login" command with support for QR code or Phone Pairing Code.
func runLogin(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "GhostWA v2.5.8 Device Linking")
	fmt.Fprintln(stdout, "──────────────────────────────────────────────────────")
	fmt.Fprintln(stdout)

	phoneNum := ""
	for i, arg := range args {
		if arg == "--phone" || arg == "-p" {
			if i+1 < len(args) {
				phoneNum = args[i+1]
			}
		} else if !strings.HasPrefix(arg, "-") && phoneNum == "" {
			phoneNum = arg
		}
	}

	// Interactive mode if no phone flag passed
	mode := "qr"
	if phoneNum != "" {
		mode = "phone"
	} else if len(args) == 0 {
		fmt.Fprintln(stdout, "Choose your authentication method:")
		fmt.Fprintln(stdout, "  [1] Scan QR Code (default)")
		fmt.Fprintln(stdout, "  [2] Link with Phone Number Pairing Code")
		fmt.Fprint(stdout, "\nSelect option [1/2]: ")

		r := bufio.NewReader(os.Stdin)
		choice, _ := r.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice == "2" {
			mode = "phone"
			fmt.Fprint(stdout, "Enter your phone number with country code (e.g. 919876543210): ")
			phoneInput, _ := r.ReadString('\n')
			phoneNum = strings.TrimSpace(phoneInput)
			if phoneNum == "" {
				fmt.Fprintln(stderr, "Error: phone number cannot be empty.")
				return 1
			}
		}
	}

	// Connect to or start the daemon process
	conn, err := wadaemon.ConnectOrStartDaemon()
	if err != nil {
		fmt.Fprintf(stderr, "Error connecting to daemon: %v\n", err)
		return 1
	}
	defer conn.Close()

	if mode == "phone" {
		phoneNum = strings.TrimPrefix(phoneNum, "+")
		req := wadaemon.Request{
			Type: "login_phone",
			Body: phoneNum,
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
			fmt.Fprintf(stderr, "Phone pairing error: %s\n", resp.Error)
			return 1
		}

		code := resp.Code
		formattedCode := code
		if len(code) == 8 {
			formattedCode = fmt.Sprintf("%s - %s", code[:4], code[4:])
		}

		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "  ┌────────────────────────────────────────────────────────┐")
		fmt.Fprintf(stdout, "  │   YOUR PAIRING CODE:   \033[1;32m%-25s\033[0m   │\n", formattedCode)
		fmt.Fprintln(stdout, "  └────────────────────────────────────────────────────────┘")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "To complete linking:")
		fmt.Fprintln(stdout, "  1. Open WhatsApp on your phone")
		fmt.Fprintln(stdout, "  2. Go to Settings → Linked Devices → Link a Device")
		fmt.Fprintln(stdout, "  3. Tap 'Link with phone number instead'")
		fmt.Fprintf(stdout, "  4. Enter code: %s\n\n", formattedCode)
		fmt.Fprintln(stdout, "Waiting for authentication on phone...")

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return 0
			}
			var evt wadaemon.Event
			if err := json.Unmarshal(line, &evt); err == nil && evt.Type == "login_success" {
				fmt.Fprintln(stdout, "\n✓ Successfully linked device via Phone Pairing Code!")
				return 0
			}
		}
	}

	// Default QR Code Login
	req := wadaemon.Request{Type: "login"}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	if err != nil {
		fmt.Fprintf(stderr, "Error writing login request to daemon: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Scan this QR code using:")
	fmt.Fprintln(stdout, "WhatsApp → Settings → Linked Devices → Link a Device")
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
				fmt.Fprintln(stdout, "Waiting for QR scan...")
			case "login_success":
				fmt.Fprintln(stdout)
				fmt.Fprintln(stdout, "✓ Successfully linked device!")
				fmt.Fprintln(stdout, "✓ Session saved")
				return 0
			case "error":
				fmt.Fprintf(stderr, "Pairing error encountered.\n")
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
