package cli

import (
	"fmt"
	"io"
)

// printStatus writes a snapshot of the client's state.
//
// Phase 1 has no WhatsApp connection yet, so these values are placeholders.
// Later phases will fill them from a real session.
func printStatus(w io.Writer) {
	fmt.Fprintln(w, "WACLI Status")
	fmt.Fprintln(w, "────────────────────────")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Logged in : no")
	fmt.Fprintln(w, "Connected : no")
	fmt.Fprintln(w, "User      : —")
}
