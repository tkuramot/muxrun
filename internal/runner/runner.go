package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/tmux"
)

var (
	ErrGroupNotFound = errors.New("group not found")
	ErrAppNotFound   = errors.New("app not found")
)

type UpOptions struct {
	GroupName   string
	AppName     string
	DirOverride string
	Force       bool
}

type DownOptions struct {
	GroupName string
	AppName   string
}

type AppStatus struct {
	Group  string
	App    string
	Status Status
	PID    int
	Dir    string
}

type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
)

type Runner struct {
	cfg  *config.Config
	tmux tmux.Client
}

func New(cfg *config.Config, tmuxClient tmux.Client) *Runner {
	return &Runner{
		cfg:  cfg,
		tmux: tmuxClient,
	}
}

func (r *Runner) Up(ctx context.Context, opts UpOptions) error {
	groups := r.cfg.FindGroups(opts.GroupName)
	if opts.GroupName != "" && len(groups) == 0 {
		return fmt.Errorf("%w: %s", ErrGroupNotFound, opts.GroupName)
	}

	for _, g := range groups {
		var apps []config.App
		if opts.AppName != "" {
			app := g.FindApp(opts.AppName)
			if app == nil {
				return fmt.Errorf("%w: %s in group %s", ErrAppNotFound, opts.AppName, g.Name)
			}
			apps = []config.App{*app}
		} else {
			apps = g.Apps
		}

		session := tmux.SessionName(g.Name)
		if err := tmux.EnsureSession(r.tmux, session); err != nil {
			return fmt.Errorf("creating session for group %q: %w", g.Name, err)
		}

		dir := g.Dir
		if opts.DirOverride != "" {
			dir = opts.DirOverride
		}

		for _, app := range apps {

			// Check if already running
			hasWin, err := tmux.HasWindow(r.tmux, session, app.Name)
			if err == nil && hasWin {
				if opts.Force {
					if err := r.tmux.KillWindow(session, app.Name); err != nil {
						return fmt.Errorf("killing window for app %q: %w", app.Name, err)
					}
				} else {
					return fmt.Errorf("%w: %s/%s", tmux.ErrAppAlreadyRunning, g.Name, app.Name)
				}
			}

			if err := r.tmux.NewWindow(session, app.Name, dir); err != nil {
				return fmt.Errorf("creating window for app %q: %w", app.Name, err)
			}

			if err := r.tmux.SendKeys(session, app.Name, app.Cmd); err != nil {
				return fmt.Errorf("sending command for app %q: %w", app.Name, err)
			}

			fmt.Printf("started %s/%s\n", g.Name, app.Name)
		}

		// Clean up the default window created with the session
		r.cleanupDefaultWindow(session)
	}

	return nil
}

func (r *Runner) Down(ctx context.Context, opts DownOptions) error {
	groups := r.cfg.FindGroups(opts.GroupName)
	if opts.GroupName != "" && len(groups) == 0 {
		return fmt.Errorf("%w: %s", ErrGroupNotFound, opts.GroupName)
	}

	for _, g := range groups {
		session := tmux.SessionName(g.Name)

		exists, err := r.tmux.HasSession(session)
		if err != nil || !exists {
			continue
		}

		if opts.AppName != "" {
			app := g.FindApp(opts.AppName)
			if app == nil {
				return fmt.Errorf("%w: %s in group %s", ErrAppNotFound, opts.AppName, g.Name)
			}

			// Best-effort check; ignore error since we're tearing down
			hasWin, _ := tmux.HasWindow(r.tmux, session, app.Name)
			if !hasWin {
				continue // already stopped, not an error
			}

			if err := r.tmux.KillWindow(session, app.Name); err != nil {
				return fmt.Errorf("killing window for app %q: %w", app.Name, err)
			}
			fmt.Printf("stopped %s/%s\n", g.Name, app.Name)

			// If no more windows, clean up the session (best-effort)
			windows, _ := r.tmux.ListWindows(session)
			if len(windows) == 0 {
				_ = r.tmux.KillSession(session)
			}
		} else {
			if err := r.tmux.KillSession(session); err != nil {
				return fmt.Errorf("killing session for group %q: %w", g.Name, err)
			}
			fmt.Printf("stopped group %s\n", g.Name)
		}
	}

	return nil
}

func (r *Runner) Status() ([]AppStatus, error) {
	var statuses []AppStatus

	for _, g := range r.cfg.Groups {
		session := tmux.SessionName(g.Name)
		exists, err := r.tmux.HasSession(session)
		if err != nil {
			continue // skip group on tmux query failure
		}

		var windows []tmux.Window
		if exists {
			windows, err = r.tmux.ListWindows(session)
			if err != nil {
				continue // skip group on tmux query failure
			}
		}

		for _, app := range g.Apps {
			s := AppStatus{
				Group:  g.Name,
				App:    app.Name,
				Status: StatusStopped,
			}

			if exists {
				for _, w := range windows {
					if w.Name == app.Name {
						s.Status = StatusRunning
						s.PID = w.PID
						s.Dir = w.Dir
						break
					}
				}
			}

			statuses = append(statuses, s)
		}
	}

	return statuses, nil
}

func (r *Runner) cleanupDefaultWindow(session string) {
	windows, err := r.tmux.ListWindows(session)
	if err != nil || len(windows) <= 1 {
		return
	}
	// The default window created by new-session is typically at index 0
	// We only remove it if there are other windows and it's unnamed or has default name
	for _, w := range windows {
		if w.Name == "zsh" || w.Name == "bash" || w.Name == "sh" || w.Name == "fish" {
			_ = r.tmux.KillWindow(session, w.Name) // best-effort cleanup
			return
		}
	}
}

