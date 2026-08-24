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
	killTimeout  = 5 * time.Second
	pollInterval = 20 * time.Millisecond
)

// TerminateProcess sends SIGTERM to a process, then SIGKILL after timeout.
func TerminateProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil // process doesn't exist
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return nil // already dead
	}

	// The target is not a child of this process (it was spawned by another
	// muxrun invocation), so proc.Wait() cannot report its exit — it returns
	// "no child processes" immediately. Poll for the process to disappear
	// instead, otherwise the SIGKILL below would never be reached. (A caller
	// that is the parent must reap the process; a zombie still answers
	// signal 0.)
	if waitForExit(pid, killTimeout) {
		return nil
	}

	if err := proc.Signal(syscall.SIGKILL); err != nil {
		return nil
	}
	waitForExit(pid, killTimeout)
	return nil
}

// waitForExit polls until the process is gone, reporting whether it exited
// before the timeout.
func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if !IsProcessRunning(pid) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(pollInterval)
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
