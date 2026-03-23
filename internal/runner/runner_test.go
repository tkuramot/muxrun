package runner

import (
	"errors"
	"testing"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/tmux"
)

func testConfig() *config.Config {
	return &config.Config{
		Groups: []config.Group{
			{
				Name: "backend",
				Dir:  "/tmp",
				Apps: []config.App{
					{Name: "api", Cmd: "echo api"},
					{Name: "worker", Cmd: "echo worker"},
				},
			},
			{
				Name: "frontend",
				Dir:  "/tmp",
				Apps: []config.App{
					{Name: "dev", Cmd: "echo dev"},
				},
			},
		},
	}
}

func TestUp_AllGroups(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	err := r.Up(UpOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check sessions were created
	if _, ok := mock.Sessions["muxrun-backend"]; !ok {
		t.Error("expected muxrun-backend session")
	}
	if _, ok := mock.Sessions["muxrun-frontend"]; !ok {
		t.Error("expected muxrun-frontend session")
	}
}

func TestUp_SpecificGroup(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	err := r.Up(UpOptions{GroupName: "backend"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := mock.Sessions["muxrun-backend"]; !ok {
		t.Error("expected muxrun-backend session")
	}
	if _, ok := mock.Sessions["muxrun-frontend"]; ok {
		t.Error("did not expect muxrun-frontend session")
	}
}

func TestUp_SpecificApp(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	err := r.Up(UpOptions{GroupName: "backend", AppName: "api"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	windows := mock.Sessions["muxrun-backend"]
	if len(windows) != 1 || windows[0].Name != "api" {
		t.Errorf("expected only api window, got %v", windows)
	}
}

func TestUp_GroupNotFound(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	err := r.Up(UpOptions{GroupName: "nonexistent"})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Errorf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestUp_AppNotFound(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	err := r.Up(UpOptions{GroupName: "backend", AppName: "nonexistent"})
	if !errors.Is(err, ErrAppNotFound) {
		t.Errorf("expected ErrAppNotFound, got %v", err)
	}
}

func TestUp_AlreadyRunning(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	r.Up(UpOptions{GroupName: "backend", AppName: "api"})
	err := r.Up(UpOptions{GroupName: "backend", AppName: "api"})
	if !errors.Is(err, tmux.ErrAppAlreadyRunning) {
		t.Errorf("expected ErrAppAlreadyRunning, got %v", err)
	}
}

func TestDown_SpecificGroup(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	r.Up(UpOptions{GroupName: "backend"})
	err := r.Down(DownOptions{GroupName: "backend"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := mock.Sessions["muxrun-backend"]; ok {
		t.Error("expected muxrun-backend session to be removed")
	}
}

func TestDown_StoppedApp(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	// Down on something that's not running should succeed silently
	err := r.Down(DownOptions{GroupName: "backend", AppName: "api"})
	if err != nil {
		t.Errorf("expected no error for stopped app, got %v", err)
	}
}

func TestStatus(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)

	r.Up(UpOptions{GroupName: "backend"})

	statuses, err := r.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statuses) != 3 {
		t.Fatalf("expected 3 statuses, got %d", len(statuses))
	}

	// backend apps should be running
	for _, s := range statuses {
		if s.Group == "backend" {
			if s.Status != StatusRunning {
				t.Errorf("expected %s/%s to be running", s.Group, s.App)
			}
		} else {
			if s.Status != StatusStopped {
				t.Errorf("expected %s/%s to be stopped", s.Group, s.App)
			}
		}
	}
}
