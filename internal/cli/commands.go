package cli

import (
	"fmt"
	"io"
)

// runCommands lists all available CLI command operations for the user.
func runCommands(stdout io.Writer) int {
	fmt.Fprintln(stdout, "Available GhostWA v2.5.4 Commands:")
	fmt.Fprintln(stdout, "──────────────────────────────────────────────────────────────────")
	fmt.Fprintln(stdout, "  ghostwa login                       Link your phone by scanning a QR code")
	fmt.Fprintln(stdout, "  ghostwa logout                      Unlink your device and clear session")
	fmt.Fprintln(stdout, "  ghostwa status                      Show WhatsApp sync and connection status")
	fmt.Fprintln(stdout, "  ghostwa chats                       Show all direct chat conversations")
	fmt.Fprintln(stdout, "  ghostwa groups                      Show all group conversations")
	fmt.Fprintln(stdout, "  ghostwa contacts                    Show list of registered contacts")
	fmt.Fprintln(stdout, "  ghostwa add-contact <name> <phone>  Register a friendly contact name manually")
	fmt.Fprintln(stdout, "  ghostwa open <chat>                 Open interactive conversation loop (direct/group)")
	fmt.Fprintln(stdout, "  ghostwa search <text>               Search contacts/group names & message texts")
	fmt.Fprintln(stdout, "  ghostwa send <to> <text|file>       Send text message or local media file")
	fmt.Fprintln(stdout, "  ghostwa delete-chat <recipient>     Delete a conversation and message history")
	fmt.Fprintln(stdout, "  ghostwa sync <chats|history>        Manually trigger contact/group or history sync")
	fmt.Fprintln(stdout, "  ghostwa listen                      Listen for real-time WhatsApp incoming messages")
	fmt.Fprintln(stdout, "  ghostwa show                        Launch the full Terminal User Interface Dashboard")
	fmt.Fprintln(stdout, "  ghostwa daemon <start|stop|restart> Control background daemon")
	fmt.Fprintln(stdout, "──────────────────────────────────────────────────────────────────")
	return 0
}
