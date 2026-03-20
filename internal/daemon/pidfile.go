package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tkuramot/muxrun/internal/runner"
)

func pidDir() string {
	return filepath.Join(os.TempDir(), "muxrun")
}

func PIDPath(group string) string {
	return filepath.Join(pidDir(), fmt.Sprintf("daemon-%s.pid", group))
}

func WritePID(group string, pid int) error {
	if err := os.MkdirAll(pidDir(), 0o755); err != nil {
		return fmt.Errorf("creating pid directory: %w", err)
	}
	return os.WriteFile(PIDPath(group), []byte(strconv.Itoa(pid)), 0o644)
}

func ReadPID(group string) (int, error) {
	data, err := os.ReadFile(PIDPath(group))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func RemovePID(group string) error {
	return os.Remove(PIDPath(group))
}

func IsRunning(group string) bool {
	pid, err := ReadPID(group)
	if err != nil {
		return false
	}
	return runner.IsProcessRunning(pid)
}

func StopDaemon(group string) error {
	pid, err := ReadPID(group)
	if err != nil {
		// No PID file or unreadable — nothing to stop
		RemovePID(group)
		return nil
	}
	if !runner.IsProcessRunning(pid) {
		RemovePID(group)
		return nil
	}
	if err := runner.TerminateProcess(pid); err != nil {
		return fmt.Errorf("terminating daemon for group %s: %w", group, err)
	}
	RemovePID(group)
	return nil
}
