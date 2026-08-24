package daemon

import (
	"testing"
	"time"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/tmux"
)

func TestDecide(t *testing.T) {
	now := time.Unix(1700000000, 0)

	tests := []struct {
		name     string
		window   tmux.Window
		state    appState
		expected action
	}{
		{
			name:     "running app is left alone",
			window:   tmux.Window{Name: "api"},
			expected: actionNone,
		},
		{
			name:     "app that stayed up gets its budget back",
			window:   tmux.Window{Name: "api"},
			state:    appState{failures: 2, startedAt: now.Add(-healthyAfter)},
			expected: actionHealthy,
		},
		{
			name:     "app that just came back is still on probation",
			window:   tmux.Window{Name: "api"},
			state:    appState{failures: 2, startedAt: now.Add(-time.Second)},
			expected: actionNone,
		},
		{
			name:     "clean exit is not a failure",
			window:   tmux.Window{Name: "api", Dead: true},
			expected: actionNone,
		},
		{
			name:     "non-zero exit is restarted",
			window:   tmux.Window{Name: "api", Dead: true, DeadStatus: 1},
			expected: actionRestart,
		},
		{
			name:     "a Ctrl-C in the pane is left alone",
			window:   tmux.Window{Name: "api", Dead: true, DeadSignal: "int"},
			expected: actionNone,
		},
		{
			name:     "a kill is restarted",
			window:   tmux.Window{Name: "api", Dead: true, DeadSignal: "kill"},
			expected: actionRestart,
		},
		{
			name:     "backoff is respected",
			window:   tmux.Window{Name: "api", Dead: true, DeadStatus: 1},
			state:    appState{failures: 1, nextAttemptAt: now.Add(time.Second)},
			expected: actionNone,
		},
		{
			name:     "the retry budget runs out",
			window:   tmux.Window{Name: "api", Dead: true, DeadStatus: 1},
			state:    appState{failures: maxRetries},
			expected: actionGiveUp,
		},
		{
			name:     "an app given up on is not touched again",
			window:   tmux.Window{Name: "api", Dead: true, DeadStatus: 1},
			state:    appState{failures: maxRetries, gaveUp: true},
			expected: actionNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decide(now, tt.window, tt.state); got != tt.expected {
				t.Errorf("decide() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBackoff(t *testing.T) {
	tests := []struct {
		failures int
		expected time.Duration
	}{
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{6, 30 * time.Second},
		{64, 30 * time.Second},
	}
	for _, tt := range tests {
		if got := backoff(tt.failures); got != tt.expected {
			t.Errorf("backoff(%d) = %v, want %v", tt.failures, got, tt.expected)
		}
	}
}

func newTestSupervisor(t *testing.T, windows ...tmux.Window) (*supervisor, *tmux.MockClient) {
	t.Helper()

	m := tmux.NewMockClient()
	m.Sessions["muxrun-test"] = windows
	apps := []config.App{
		{Name: "api", Cmd: "go run main.go", Restart: config.RestartOnFailure},
		{Name: "once", Cmd: "echo hello"},
	}
	return newSupervisor(m, "muxrun-test", "test", "/tmp/test", apps), m
}

func kill(m *tmux.MockClient, window string, status int) {
	windows := m.Sessions["muxrun-test"]
	for i, w := range windows {
		if w.Name == window {
			windows[i].Dead = true
			windows[i].DeadStatus = status
		}
	}
}

func TestSupervisorTick_RestartsFailedApp(t *testing.T) {
	sup, m := newTestSupervisor(t, tmux.Window{Name: "api", Dead: true, DeadStatus: 1})

	sup.tick(time.Unix(1700000000, 0))

	if len(m.Respawns) != 1 {
		t.Fatalf("expected 1 respawn, got %d", len(m.Respawns))
	}
	if got := m.Respawns[0]; got.Window != "api" || got.Cmd != "go run main.go" {
		t.Errorf("unexpected respawn: %+v", got)
	}
	if got := m.Sessions["muxrun-test"][0].Restarts; got != 1 {
		t.Errorf("expected the restart count on the window to be 1, got %d", got)
	}
}

func TestSupervisorTick_IgnoresAppsWithoutPolicy(t *testing.T) {
	sup, m := newTestSupervisor(t, tmux.Window{Name: "once", Dead: true, DeadStatus: 1})

	sup.tick(time.Unix(1700000000, 0))

	if len(m.Respawns) != 0 {
		t.Errorf("expected no respawn for an app with restart = no, got %d", len(m.Respawns))
	}
}

func TestSupervisorTick_GivesUpAfterMaxRetries(t *testing.T) {
	sup, m := newTestSupervisor(t, tmux.Window{Name: "api", Dead: true, DeadStatus: 1})

	now := time.Unix(1700000000, 0)
	for i := 0; i < maxRetries+1; i++ {
		sup.tick(now)
		kill(m, "api", 1)
		now = now.Add(maxBackoff)
	}

	if len(m.Respawns) != maxRetries {
		t.Errorf("expected %d respawns, got %d", maxRetries, len(m.Respawns))
	}
	w := m.Sessions["muxrun-test"][0]
	if !w.RestartFailed {
		t.Error("expected the window to be marked failed")
	}
	if w.Restarts != maxRetries {
		t.Errorf("expected the restart count on the window to be %d, got %d", maxRetries, w.Restarts)
	}
}

func TestSupervisorTick_HonoursBackoff(t *testing.T) {
	sup, m := newTestSupervisor(t, tmux.Window{Name: "api", Dead: true, DeadStatus: 1})

	now := time.Unix(1700000000, 0)
	sup.tick(now)
	kill(m, "api", 1)
	sup.tick(now.Add(baseBackoff / 2))

	if len(m.Respawns) != 1 {
		t.Errorf("expected the second attempt to wait for the backoff, got %d respawns", len(m.Respawns))
	}
}

func TestSupervisorRestartOnChange_ResetsBudget(t *testing.T) {
	sup, m := newTestSupervisor(t, tmux.Window{Name: "api", Dead: true, DeadStatus: 1})
	sup.states["api"] = &appState{failures: maxRetries, gaveUp: true}

	if err := sup.restartOnChange("api", "go run main.go"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	st := sup.states["api"]
	if st.failures != 0 || st.gaveUp {
		t.Errorf("expected a fresh retry budget, got %+v", st)
	}
	if m.Sessions["muxrun-test"][0].RestartFailed {
		t.Error("expected the failed mark to be cleared from the window")
	}

	kill(m, "api", 1)
	sup.tick(time.Unix(1700000000, 0))
	if len(m.Respawns) != 2 {
		t.Errorf("expected the app to be restarted again, got %d respawns", len(m.Respawns))
	}
}
