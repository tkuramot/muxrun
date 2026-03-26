package tmux

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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
	KillWindow(session, window string) error
	ListWindows(session string) ([]Window, error)
	SendKeys(session, window, keys string) error
	GetPanePID(session, window string) (int, error)
	CapturePane(session, window string) (string, error)
	PipePane(session, window, cmd string) error
	UnpipePane(session, window string) error
}

type Session struct {
	Name string
}

type Window struct {
	Name string
	PID  int
	Dir  string
}

type client struct {
	tmuxPath string
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
	_, err := c.run("new-window", "-t", session, "-n", window, "-c", dir)
	return err
}

func (c *client) KillWindow(session, window string) error {
	_, err := c.run("kill-window", "-t", session+":"+window)
	return err
}

func (c *client) ListWindows(session string) ([]Window, error) {
	out, err := c.run("list-windows", "-t", session, "-F", "#{window_name} #{pane_pid} #{pane_current_path}")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var windows []Window
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 2 {
			continue
		}
		pid, _ := strconv.Atoi(parts[1])
		dir := ""
		if len(parts) == 3 {
			dir = parts[2]
		}
		windows = append(windows, Window{Name: parts[0], PID: pid, Dir: dir})
	}
	return windows, nil
}

func (c *client) SendKeys(session, window, keys string) error {
	_, err := c.run("send-keys", "-t", session+":"+window, keys, "Enter")
	return err
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
