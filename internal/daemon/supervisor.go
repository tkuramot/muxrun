package daemon

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/tmux"
)

const (
	pollInterval = time.Second
	baseBackoff  = time.Second
	maxBackoff   = 30 * time.Second
	maxRetries   = 5
	// How long an app has to stay up before its failures stop counting.
	healthyAfter = 10 * time.Second
)

// Signals muxrun reads as a deliberate stop, so restarting would fight the
// user. SIGKILL is excluded: it is more often an OOM kill. tmux reports signal
// names lowercase and without the SIG prefix.
var stopSignals = map[string]bool{"int": true, "term": true, "hup": true, "quit": true}

type appState struct {
	failures      int // consecutive; drives the retry budget and the backoff
	restarts      int // every restart since the window was created
	startedAt     time.Time
	nextAttemptAt time.Time
	gaveUp        bool
}

type action int

const (
	actionNone action = iota
	actionRestart
	actionGiveUp
	actionHealthy
)

func decide(now time.Time, w tmux.Window, st appState) action {
	if !w.Dead {
		if st.failures > 0 && !st.startedAt.IsZero() && now.Sub(st.startedAt) >= healthyAfter {
			return actionHealthy
		}
		return actionNone
	}
	if st.gaveUp || !isFailure(w) {
		return actionNone
	}
	if st.failures >= maxRetries {
		return actionGiveUp
	}
	if now.Before(st.nextAttemptAt) {
		return actionNone
	}
	return actionRestart
}

// A tmux older than 3.4 reports no signal, leaving a signalled pane looking
// like a clean exit — the quiet direction to be wrong in.
func isFailure(w tmux.Window) bool {
	if w.DeadSignal != "" {
		return !stopSignals[w.DeadSignal]
	}
	return w.DeadStatus != 0
}

func backoff(failures int) time.Duration {
	d := baseBackoff << (failures - 1)
	if d > maxBackoff || d <= 0 {
		return maxBackoff
	}
	return d
}

// supervisor restarts apps configured with restart = "on-failure". It also owns
// the watch daemon's restarts, so the two can never interleave on one window.
type supervisor struct {
	tmux    tmux.Client
	session string
	group   string
	dir     string
	apps    map[string]string // app name -> cmd, restart-enabled apps only

	mu     sync.Mutex
	states map[string]*appState
}

func newSupervisor(c tmux.Client, session, group, dir string, apps []config.App) *supervisor {
	s := &supervisor{
		tmux:    c,
		session: session,
		group:   group,
		dir:     dir,
		apps:    make(map[string]string),
		states:  make(map[string]*appState),
	}
	for _, app := range apps {
		if app.Restart != config.RestartOnFailure {
			continue
		}
		s.apps[app.Name] = app.Cmd
		s.states[app.Name] = &appState{}
	}
	return s
}

func (s *supervisor) enabled() bool {
	return len(s.apps) > 0
}

// restartOnChange restarts an app because its files changed, with a fresh retry
// budget: the code it failed with is not the code it is about to run.
func (s *supervisor) restartOnChange(app, cmd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.tmux.RespawnWindow(s.session, app, s.dir, cmd); err != nil {
		return err
	}
	if st, ok := s.states[app]; ok {
		st.failures = 0
		st.gaveUp = false
		st.startedAt = time.Now()
		st.nextAttemptAt = time.Time{}
		s.setOption(app, tmux.RestartStateOption, "")
	}
	return nil
}

func (s *supervisor) run(stop <-chan struct{}) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

func (s *supervisor) tick(now time.Time) {
	windows, err := s.tmux.ListWindows(s.session)
	if err != nil {
		return // try again next tick
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, w := range windows {
		cmd, ok := s.apps[w.Name]
		if !ok {
			continue
		}
		st := s.states[w.Name]

		switch decide(now, w, *st) {
		case actionHealthy:
			st.failures = 0
			st.nextAttemptAt = time.Time{}
			log.Printf("%s/%s is back up, retry budget reset", s.group, w.Name)

		case actionGiveUp:
			st.gaveUp = true
			s.setOption(w.Name, tmux.RestartStateOption, tmux.RestartStateFailed)
			log.Printf("giving up on %s/%s after %d restarts (%s)", s.group, w.Name, st.restarts, exitReason(w))

		case actionRestart:
			if err := s.tmux.RespawnWindow(s.session, w.Name, s.dir, cmd); err != nil {
				log.Printf("failed to restart %s/%s: %v", s.group, w.Name, err)
				continue
			}
			st.failures++
			st.restarts++
			st.startedAt = now
			st.nextAttemptAt = now.Add(backoff(st.failures))
			s.setOption(w.Name, tmux.RestartsOption, strconv.Itoa(st.restarts))
			log.Printf("restarted %s/%s (%s, attempt %d/%d)", s.group, w.Name, exitReason(w), st.failures, maxRetries)
		}
	}
}

// setOption is best-effort: losing it only costs muxrun ps some detail.
func (s *supervisor) setOption(app, option, value string) {
	if err := s.tmux.SetWindowOption(s.session, app, option, value); err != nil {
		log.Printf("failed to set %s on %s/%s: %v", option, s.group, app, err)
	}
}

func exitReason(w tmux.Window) string {
	if w.DeadSignal != "" {
		return "killed by SIG" + strings.ToUpper(w.DeadSignal)
	}
	return "exit " + strconv.Itoa(w.DeadStatus)
}
