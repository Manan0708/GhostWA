package cli

import (
	"fmt"
	"io"
)

// runCommands lists all available CLI command operations for the user.
func runCommands(stdout io.Writer) int {
	fmt.Fprintln(stdout, "Available WACLI Commands:")
	fmt.Fprintln(stdout, "────────────────────────────────────────────────────────────")
	fmt.Fprintln(stdout, "  wacli login                       Link your phone by scanning a QR code")
	fmt.Fprintln(stdout, "  wacli logout                      Unlink your device and clear session")
	fmt.Fprintln(stdout, "  wacli status                      Show WhatsApp sync and connection status")
	fmt.Fprintln(stdout, "  wacli chats                       Show all direct chat conversations")
	fmt.Fprintln(stdout, "  wacli groups                      Show all group conversations")
	fmt.Fprintln(stdout, "  wacli contacts                    Show list of registered contacts")
	fmt.Fprintln(stdout, "  wacli add-contact <name> <phone>  Register a friendly contact name manually")
	fmt.Fprintln(stdout, "  wacli open <chat> [limit]         Open interactive conversation loop (direct/group)")
	fmt.Fprintln(stdout, "  wacli search <text>               Search contacts/group names & message texts")
	fmt.Fprintln(stdout, "  wacli send <to> <text|file>       Send text message or local media file")
	fmt.Fprintln(stdout, "  wacli listen                      Listen for real-time WhatsApp incoming messages")
	fmt.Fprintln(stdout, "  wacli show                        Launch the full Terminal User Interface Dashboard")
	fmt.Fprintln(stdout, "  wacli daemon <start|stop|restart> Control background daemon")
	fmt.Fprintln(stdout, "────────────────────────────────────────────────────────────")
	return 0
}
