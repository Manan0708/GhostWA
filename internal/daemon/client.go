package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// StartDaemonProcess spawns the wacli executable as a detached background daemon.
func StartDaemonProcess() error {
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}

	cmd := exec.Command(exe, "daemon-run")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW for Windows background execution
	}
	return cmd.Start()
}

// ConnectOrStartDaemon dials the daemon port. If down, starts it automatically.
func ConnectOrStartDaemon() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:9090", 200*time.Millisecond)
	if err == nil {
		return conn, nil
	}

	// Try starting
	if err := StartDaemonProcess(); err != nil {
		return nil, fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Wait for TCP port to open (up to 3 seconds)
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err = net.DialTimeout("tcp", "127.0.0.1:9090", 100*time.Millisecond)
		if err == nil {
			return conn, nil
		}
	}
	return nil, fmt.Errorf("daemon started but failed to listen on port 9090: %w", err)
}

// StopDaemon connects to the daemon and issues the stop shutdown instruction.
func StopDaemon() error {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		return fmt.Errorf("daemon is not running: %w", err)
	}
	defer conn.Close()

	req := Request{Type: "stop"}
	data, _ := json.Marshal(req)
	_, err = conn.Write(append(data, '\n'))
	return err
}
