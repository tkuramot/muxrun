package runner

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

const (
	killTimeout = 5 * time.Second
)

// TerminateProcess sends SIGTERM to a process, then SIGKILL after timeout.
func TerminateProcess(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil // process doesn't exist
	}

	// Send SIGTERM
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return nil // already dead
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(killTimeout):
		// Force kill
		if err := proc.Signal(syscall.SIGKILL); err != nil {
			return nil
		}
		<-done
		return nil
	}
}

// IsProcessRunning checks if a process with the given PID is alive.
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// GetChildPIDs returns child process IDs. On macOS/Linux we use pgrep.
func GetChildPIDs(ppid int) ([]int, error) {
	_ = ppid
	return nil, fmt.Errorf("not implemented")
}
