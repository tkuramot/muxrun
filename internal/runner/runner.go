package runner

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/tkuramot/muxrun/internal/watcher"
)

var (
	ErrGroupNotFound = errors.New("group not found")
	ErrAppNotFound   = errors.New("app not found")
)

type UpOptions struct {
	GroupName   string
	AppName     string
	DirOverride string
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
}

type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
)

type Runner struct {
	cfg      *config.Config
	tmux     tmux.Client
	watchers []*appWatcher
}

type appWatcher struct {
	watcher   *watcher.Watcher
	debouncer *watcher.Debouncer
	session   string
	window    string
	cmd       string
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

		for _, app := range apps {
			dir := app.Dir
			if opts.DirOverride != "" {
				dir = opts.DirOverride
			}

			// Check if already running
			hasWin, err := tmux.HasWindow(r.tmux, session, app.Name)
			if err == nil && hasWin {
				return fmt.Errorf("%w: %s/%s", tmux.ErrAppAlreadyRunning, g.Name, app.Name)
			}

			if err := r.tmux.NewWindow(session, app.Name, dir); err != nil {
				return fmt.Errorf("creating window for app %q: %w", app.Name, err)
			}

			if err := r.tmux.SendKeys(session, app.Name, app.Cmd); err != nil {
				return fmt.Errorf("sending command for app %q: %w", app.Name, err)
			}

			fmt.Printf("started %s/%s\n", g.Name, app.Name)

			if app.Watch.Enabled {
				if err := r.startWatcher(session, app.Name, app.Cmd, dir, app.Watch.Exclude); err != nil {
					fmt.Printf("warning: failed to start watcher for %s/%s: %v\n", g.Name, app.Name, err)
				}
			}
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

			r.stopWatcher(session, app.Name)

			// Kill specific window
			hasWin, _ := tmux.HasWindow(r.tmux, session, app.Name)
			if !hasWin {
				continue // already stopped, not an error
			}

			if err := r.tmux.KillWindow(session, app.Name); err != nil {
				return fmt.Errorf("killing window for app %q: %w", app.Name, err)
			}
			fmt.Printf("stopped %s/%s\n", g.Name, app.Name)

			// If no more windows, kill session
			windows, _ := r.tmux.ListWindows(session)
			if len(windows) == 0 {
				r.tmux.KillSession(session)
			}
		} else {
			// Stop all watchers for this session
			r.stopSessionWatchers(session)

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
		exists, _ := r.tmux.HasSession(session)

		var windows []tmux.Window
		if exists {
			windows, _ = r.tmux.ListWindows(session)
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
						break
					}
				}
			}

			statuses = append(statuses, s)
		}
	}

	return statuses, nil
}

func (r *Runner) PIDString(pid int) string {
	if pid <= 0 {
		return "-"
	}
	return strconv.Itoa(pid)
}

func (r *Runner) startWatcher(session, window, cmd, dir string, excludePatterns []string) error {
	w, err := watcher.New(dir, excludePatterns)
	if err != nil {
		return err
	}

	aw := &appWatcher{
		watcher: w,
		session: session,
		window:  window,
		cmd:     cmd,
	}

	aw.debouncer = watcher.NewDebouncer(500*time.Millisecond, func() {
		r.restartApp(session, window, cmd)
	})

	r.watchers = append(r.watchers, aw)

	go func() {
		for range w.Events() {
			aw.debouncer.Trigger()
		}
	}()

	return nil
}

func (r *Runner) restartApp(session, window, cmd string) {
	// Send Ctrl+C to stop current process, then re-run command
	r.tmux.SendKeys(session, window, "C-c")
	time.Sleep(100 * time.Millisecond)
	r.tmux.SendKeys(session, window, cmd)
	fmt.Printf("restarted %s:%s\n", tmux.GroupName(session), window)
}

func (r *Runner) stopWatcher(session, window string) {
	for i, aw := range r.watchers {
		if aw.session == session && aw.window == window {
			aw.debouncer.Stop()
			aw.watcher.Stop()
			r.watchers = append(r.watchers[:i], r.watchers[i+1:]...)
			return
		}
	}
}

func (r *Runner) stopSessionWatchers(session string) {
	remaining := r.watchers[:0]
	for _, aw := range r.watchers {
		if aw.session == session {
			aw.debouncer.Stop()
			aw.watcher.Stop()
		} else {
			remaining = append(remaining, aw)
		}
	}
	r.watchers = remaining
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
			r.tmux.KillWindow(session, w.Name)
			return
		}
	}
}

func (r *Runner) StopAllWatchers() {
	for _, aw := range r.watchers {
		aw.debouncer.Stop()
		aw.watcher.Stop()
	}
	r.watchers = nil
}
