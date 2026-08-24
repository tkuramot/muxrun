package tmux

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const InitWindowName = "__muxrun_init__"

var (
	ErrTmuxNotAvailable = errors.New("tmux is not available")
)

type Client interface {
	HasSession(name string) (bool, error)
	NewSession(name string) error
	KillSession(name string) error
	ListSessions() ([]Session, error)
	NewWindow(session, window, dir string) error
	SetWindowOption(session, window, option, value string) error
	RespawnWindow(session, window, dir, cmd string) error
	KillWindow(session, window string) error
	ListWindows(session string) ([]Window, error)
	GetPanePID(session, window string) (int, error)
	CapturePane(session, window string) (string, error)
	PipePane(session, window, cmd string) error
	UnpipePane(session, window string) error
}

type Session struct {
	Name string
}

type Window struct {
	Name          string
	PID           int
	Dir           string
	Dead          bool
	DeadStatus    int
	DeadSignal    string
	DeadTime      time.Time
	Restarts      int
	RestartFailed bool
}

// Window options muxrun sets on its own windows. They die with the window, so
// `muxrun up` resets them.
const (
	RestartsOption     = "@muxrun_restarts"
	RestartStateOption = "@muxrun_restart_state"
	RestartStateFailed = "failed"
)

type client struct {
	tmuxPath string
	shell    string
}

func NewClient() (Client, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, ErrTmuxNotAvailable
	}
	return &client{tmuxPath: path}, nil
}

func (c *client) run(args ...string) (string, error) {
	cmd := exec.Command(c.tmuxPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (c *client) HasSession(name string) (bool, error) {
	cmd := exec.Command(c.tmuxPath, "has-session", "-t", name)
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c *client) NewSession(name string) error {
	_, err := c.run("new-session", "-d", "-s", name, "-n", InitWindowName)
	return err
}

func (c *client) KillSession(name string) error {
	_, err := c.run("kill-session", "-t", name)
	return err
}

func (c *client) ListSessions() ([]Session, error) {
	out, err := c.run("list-sessions", "-F", "#{session_name}")
	if err != nil {
		if strings.Contains(err.Error(), "no server running") || strings.Contains(err.Error(), "no sessions") {
			return nil, nil
		}
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var sessions []Session
	for _, line := range strings.Split(out, "\n") {
		if line != "" {
			sessions = append(sessions, Session{Name: line})
		}
	}
	return sessions, nil
}

func (c *client) NewWindow(session, window, dir string) error {
	if _, err := c.run("new-window", "-t", session, "-n", window, "-c", dir); err != nil {
		return err
	}
	_, err := c.run("set-option", "-t", session+":"+window, "remain-on-exit", "on")
	return err
}

// SetWindowOption sets a window option. An empty value clears it.
func (c *client) SetWindowOption(session, window, option, value string) error {
	if value == "" {
		_, err := c.run("set-option", "-w", "-t", session+":"+window, "-u", option)
		return err
	}
	_, err := c.run("set-option", "-w", "-t", session+":"+window, option, value)
	return err
}

func (c *client) KillWindow(session, window string) error {
	_, err := c.run("kill-window", "-t", session+":"+window)
	return err
}

// RespawnWindow restarts the window's pane with cmd, killing whatever is
// running there first. Unlike SendKeys it also works on a pane left dead by
// remain-on-exit, and the command becomes the pane process itself rather than
// a child of an interactive shell.
func (c *client) RespawnWindow(session, window, dir, cmd string) error {
	args := []string{"respawn-window", "-k", "-t", session + ":" + window}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	args = append(args, c.shellCommand(cmd))
	_, err := c.run(args...)
	return err
}

// loginShells are the shells muxrun knows how to start as an interactive
// login shell. Other shells are given -c only, since -l and -i are not
// portable across every /bin/sh.
var loginShells = map[string]bool{"bash": true, "zsh": true, "fish": true}

// shellCommand wraps cmd so it runs with the environment it used to get when
// muxrun typed the command into the pane's interactive login shell: rc files,
// PATH set up by version managers, aliases. exec avoids an extra process
// layer, keeping pane_pid on the command itself.
func (c *client) shellCommand(cmd string) string {
	shell := c.defaultShell()
	flags := "-c"
	if loginShells[filepath.Base(shell)] {
		flags = "-lic"
	}
	return fmt.Sprintf("exec %s %s %s", shellQuote(shell), flags, shellQuote(cmd))
}

// defaultShell reports the shell tmux would use for a new pane, which is the
// shell muxrun commands used to run under.
func (c *client) defaultShell() string {
	if c.shell != "" {
		return c.shell
	}
	if out, err := c.run("show-option", "-gv", "default-shell"); err == nil && out != "" {
		c.shell = out
	} else if sh := os.Getenv("SHELL"); sh != "" {
		c.shell = sh
	} else {
		c.shell = "/bin/sh"
	}
	return c.shell
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (c *client) ListWindows(session string) ([]Window, error) {
	out, err := c.run("list-windows", "-t", session, "-F", windowFormat)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var windows []Window
	for _, line := range strings.Split(out, "\n") {
		if w, ok := parseWindowLine(line); ok {
			windows = append(windows, w)
		}
	}
	return windows, nil
}

// windowFormat is the list-windows format used by ListWindows. Fields are
// tab-separated because pane_current_path (and window names) may contain
// spaces, and the path is last so that a stray separator inside it cannot
// shift the fields that decide whether a pane is dead.
//
// pane_dead_status is empty when the pane process was killed by a signal, so
// pane_dead_signal is what tells a signalled pane apart from a clean exit 0.
// It only exists in tmux 3.4+; older versions expand it to an empty field, as
// they do for an unset @muxrun option.
const windowFormat = "#{window_name}\t#{pane_pid}\t#{pane_dead}\t#{pane_dead_status}\t#{pane_dead_signal}\t#{pane_dead_time}\t#{" + RestartsOption + "}\t#{" + RestartStateOption + "}\t#{pane_current_path}"

// parseWindowLine parses a single windowFormat line. Trailing fields are
// optional: a tmux that does not know one of the format variables expands it
// to an empty string, which run() may then trim away.
func parseWindowLine(line string) (Window, bool) {
	parts := strings.Split(line, "\t")
	if len(parts) < 2 {
		return Window{}, false
	}
	w := Window{Name: parts[0]}
	w.PID, _ = strconv.Atoi(parts[1])
	if len(parts) >= 3 {
		w.Dead = parts[2] == "1"
	}
	if len(parts) >= 4 {
		w.DeadStatus, _ = strconv.Atoi(parts[3])
	}
	if len(parts) >= 5 {
		w.DeadSignal = parts[4]
	}
	if len(parts) >= 6 {
		if ts, err := strconv.ParseInt(parts[5], 10, 64); err == nil && ts > 0 {
			w.DeadTime = time.Unix(ts, 0)
		}
	}
	if len(parts) >= 7 {
		w.Restarts, _ = strconv.Atoi(parts[6])
	}
	if len(parts) >= 8 {
		w.RestartFailed = parts[7] == RestartStateFailed
	}
	if len(parts) >= 9 {
		w.Dir = parts[8]
	}
	return w, true
}

func (c *client) GetPanePID(session, window string) (int, error) {
	out, err := c.run("display-message", "-t", session+":"+window, "-p", "#{pane_pid}")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

func (c *client) CapturePane(session, window string) (string, error) {
	cmd := exec.Command(c.tmuxPath, "capture-pane", "-t", session+":"+window, "-p", "-S", "-", "-J")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux capture-pane: %s", strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func (c *client) PipePane(session, window, cmd string) error {
	_, err := c.run("pipe-pane", "-t", session+":"+window, cmd)
	return err
}

func (c *client) UnpipePane(session, window string) error {
	_, err := c.run("pipe-pane", "-t", session+":"+window)
	return err
}
