//go:build !windows

package daemon

import (
	"os/exec"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// No-op for POSIX systems (Linux / macOS)
}
