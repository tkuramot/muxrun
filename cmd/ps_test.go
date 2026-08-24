package cmd

import (
	"testing"
	"time"

	"github.com/tkuramot/muxrun/internal/runner"
)

func TestFormatStatus(t *testing.T) {
	exitedAt := time.Now().Add(-3 * time.Second)

	tests := []struct {
		name     string
		status   runner.AppStatus
		expected string
	}{
		{
			name:     "running",
			status:   runner.AppStatus{Status: runner.StatusRunning},
			expected: "running",
		},
		{
			name:     "running after a restart",
			status:   runner.AppStatus{Status: runner.StatusRunning, Restarts: 1},
			expected: "running (1 restart)",
		},
		{
			name:     "exited",
			status:   runner.AppStatus{Status: runner.StatusStopped, Exited: true, ExitStatus: 1, ExitedAt: exitedAt},
			expected: "exited (1) 3s ago",
		},
		{
			name:     "killed by a signal",
			status:   runner.AppStatus{Status: runner.StatusStopped, Exited: true, ExitSignal: "int", ExitedAt: exitedAt},
			expected: "exited (SIGINT) 3s ago",
		},
		{
			name: "given up on",
			status: runner.AppStatus{
				Status: runner.StatusStopped, Exited: true, ExitStatus: 1,
				ExitedAt: exitedAt, Restarts: 5, RestartFailed: true,
			},
			expected: "failed (1) 3s ago (5 restarts)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatStatus(tt.status); got != tt.expected {
				t.Errorf("formatStatus() = %q, want %q", got, tt.expected)
			}
		})
	}
}
