package runner

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
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

// FirstChildPID returns the PID of the first direct child of the given parent.
// Returns 0 if there are no children or the lookup fails — so a tmux pane shell
// with no command running yet reports 0 instead of the shell PID itself.
func FirstChildPID(ppid int) int {
	if ppid <= 0 {
		return 0
	}
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(ppid)).Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && pid > 0 {
			return pid
		}
	}
	return 0
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