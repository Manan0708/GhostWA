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
	case "status":
		printStatus(stdout)
		return 0
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
	fmt.Fprintln(w, "  wacli status       Show login and connection status")
}
