package runner

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tkuramot/muxrun/internal/tmux"
)

func TestLogs_Output(t *testing.T) {
	mock := tmux.NewMockClient()
	mock.Sessions["muxrun-backend"] = []tmux.Window{{Name: "api"}}
	mock.CapturePaneOutput["muxrun-backend:api"] = "line1\nline2\n"

	r := New(testConfig(), mock)
	var buf bytes.Buffer
	err := r.Logs(context.Background(), LogsOptions{
		GroupName: "backend",
		AppName:   "api",
		Writer:    &buf,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "line1") {
		t.Errorf("expected output to contain 'line1', got: %q", buf.String())
	}
}

func TestLogs_StoppedAppSkipped(t *testing.T) {
	mock := tmux.NewMockClient()
	// session exists but window does not

	r := New(testConfig(), mock)
	var buf bytes.Buffer
	err := r.Logs(context.Background(), LogsOptions{
		GroupName: "backend",
		AppName:   "api",
		Writer:    &buf,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for stopped app, got: %q", buf.String())
	}
}

func TestLogs_GroupNotFound(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)
	var buf bytes.Buffer
	err := r.Logs(context.Background(), LogsOptions{
		GroupName: "nonexistent",
		AppName:   "api",
		Writer:    &buf,
	})
	if !errors.Is(err, ErrGroupNotFound) {
		t.Errorf("expected ErrGroupNotFound, got: %v", err)
	}
}

func TestLogs_AppNotFound(t *testing.T) {
	mock := tmux.NewMockClient()
	r := New(testConfig(), mock)
	var buf bytes.Buffer
	err := r.Logs(context.Background(), LogsOptions{
		GroupName: "backend",
		AppName:   "nonexistent",
		Writer:    &buf,
	})
	if !errors.Is(err, ErrAppNotFound) {
		t.Errorf("expected ErrAppNotFound, got: %v", err)
	}
}

func TestLogs_Follow_ShowsExistingOutput(t *testing.T) {
	mock := tmux.NewMockClient()
	mock.Sessions["muxrun-backend"] = []tmux.Window{{Name: "api"}}
	mock.CapturePaneOutput["muxrun-backend:api"] = "existing1\nexisting2\n"

	r := New(testConfig(), mock)
	var buf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so followLogs exits after emitting existing content

	err := r.Logs(ctx, LogsOptions{
		GroupName: "backend",
		AppName:   "api",
		Follow:    true,
		Writer:    &buf,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "existing1") {
		t.Errorf("expected existing output 'existing1', got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "existing2") {
		t.Errorf("expected existing output 'existing2', got: %q", buf.String())
	}
}
