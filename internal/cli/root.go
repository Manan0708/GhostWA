// Package cli implements the command-line interface.
//
// cmd/wacli/main.go only starts the process. All command handling lives here
// so we can test it without launching a real terminal session.
package cli

import (
	"fmt"
	"io"
)

// Run looks at the command-line arguments and dispatches to the right command.
//
// args should NOT include the program name. If the user typed:
//
//	wacli status
//
// then args is []string{"status"}.
//
// stdout is where normal output goes (what the user should see).
// stderr is where errors go. Separating them is a Unix/Windows convention:
// you can still redirect "real" output without mixing in error text.
//
// The return value is the process exit code: 0 means success, non-zero means failure.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "WACLI")
		return 0
	}

	switch args[0] {
	case "--help", "-h", "help":
		printHelp(stdout)
		return 0
	case "login":
		return runLogin(stdout, stderr)
	case "status":
		return runStatus(stdout, stderr)
	case "chats":
		return runChats(stdout, stderr)
	case "add-contact":
		return runAddContact(args[1:], stdout, stderr)
	case "contacts":
		return runContacts(stdout, stderr)
	case "commands":
		return runCommands(stdout)
	case "daemon":
		return runDaemonCLI(args[1:], stdout, stderr)
	case "daemon-run":
		return runDaemonRun(stdout, stderr)
	case "groups":
		return runGroups(stdout, stderr)
	case "logout":
		return runLogout(stdout, stderr)
	case "open":
		return runOpen(args[1:], stdout, stderr)
	case "show":
		return runShow(stdout, stderr)
	case "search":
		return runSearch(args[1:], stdout, stderr)
	case "send":
		return runSend(args[1:], stdout, stderr)
	case "listen":
		return runListen(stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		fmt.Fprintln(stderr, "Run 'wacli --help' for usage.")
		return 1
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "WACLI — lightweight WhatsApp terminal client")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  wacli              Print WACLI")
	fmt.Fprintln(w, "  wacli --help       Show this help")
	fmt.Fprintln(w, "  wacli login        Link a WhatsApp device by scanning QR code")
	fmt.Fprintln(w, "  wacli logout       Unlink your WhatsApp device and delete session")
	fmt.Fprintln(w, "  wacli status       Show login and connection status")
	fmt.Fprintln(w, "  wacli chats        Show all direct chat conversations")
	fmt.Fprintln(w, "  wacli groups       Show all group conversations")
	fmt.Fprintln(w, "  wacli contacts     Show list of registered contacts")
	fmt.Fprintln(w, "  wacli commands     List all available commands")
	fmt.Fprintln(w, "  wacli show         Launch the interactive TUI Dashboard")
	fmt.Fprintln(w, "  wacli daemon <cmd> Manage the background daemon (start|stop|restart)")
	fmt.Fprintln(w, "  wacli add-contact <name> <phone> Register a friend's contact name")
	fmt.Fprintln(w, "  wacli open <chat> [limit] Open an interactive conversation (limit defaults to 3)")
	fmt.Fprintln(w, "  wacli search <text> Search chat names and message contents")
	fmt.Fprintln(w, "  wacli send <to> <msg> Send a text message or local media file")
	fmt.Fprintln(w, "  wacli listen       Listen for incoming text messages in real-time")
}
