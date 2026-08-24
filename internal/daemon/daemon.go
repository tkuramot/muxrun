package daemon

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/tkuramot/muxrun/internal/watcher"
)

func Spawn(configPath, groupName string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	cmd := exec.Command(exe, "_daemon", "--config", configPath, "--group", groupName)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return fmt.Errorf("opening /dev/null: %w", err)
	}
	defer devNull.Close()

	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	if err := WritePID(groupName, cmd.Process.Pid); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}

	return nil
}

func Run(configPath, groupName string) error {
	// Set up log file
	logPath := filepath.Join(os.TempDir(), "muxrun", fmt.Sprintf("daemon-%s.log", groupName))
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	defer RemovePID(groupName)

	log.Printf("daemon started for group %s (pid %d)", groupName, os.Getpid())

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}

	groups := cfg.FindGroups(groupName)
	if len(groups) == 0 {
		return fmt.Errorf("group not found: %s", groupName)
	}
	group := groups[0]

	tmuxClient, err := tmux.NewClient()
	if err != nil {
		return fmt.Errorf("creating tmux client: %w", err)
	}

	session := tmux.SessionName(groupName)
	sup := newSupervisor(tmuxClient, session, groupName, group.Dir, group.Apps)

	var watchers []*watcher.Watcher
	var debouncers []*watcher.Debouncer

	for _, app := range group.Apps {
		if !app.Watch.Enabled {
			continue
		}

		w, err := watcher.New(group.Dir, app.Watch.Exclude)
		if err != nil {
			log.Printf("warning: failed to start watcher for %s/%s: %v", groupName, app.Name, err)
			continue
		}
		watchers = append(watchers, w)

		appName := app.Name
		appCmd := app.Cmd
		d := watcher.NewDebouncer(500*time.Millisecond, func() {
			hasWin, err := tmux.HasWindow(tmuxClient, session, appName)
			if err != nil || !hasWin {
				return
			}
			// respawn-window kills the running process and starts the command
			// again in one step. It also revives a pane that remain-on-exit
			// left dead, which send-keys silently fails to do.
			if err := sup.restartOnChange(appName, appCmd); err != nil {
				log.Printf("failed to restart %s/%s: %v", groupName, appName, err)
				return
			}
			log.Printf("restarted %s/%s", groupName, appName)
		})
		debouncers = append(debouncers, d)

		go func() {
			for range w.Events() {
				d.Trigger()
			}
		}()

		log.Printf("watching %s/%s (dir: %s)", groupName, appName, group.Dir)
	}

	if len(watchers) == 0 && !sup.enabled() {
		log.Printf("no watch- or restart-enabled apps found for group %s, exiting", groupName)
		return nil
	}

	stop := make(chan struct{})
	if sup.enabled() {
		go sup.run(stop)
		log.Printf("supervising restart-on-failure apps for group %s", groupName)
	}

	// Wait for signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	log.Printf("shutting down daemon for group %s", groupName)

	close(stop)
	for _, d := range debouncers {
		d.Stop()
	}
	for _, w := range watchers {
		w.Stop()
	}

	return nil
}
