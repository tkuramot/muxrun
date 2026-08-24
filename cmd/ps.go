package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tkuramot/muxrun/internal/runner"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/tkuramot/muxrun/internal/ui"
	"github.com/urfave/cli/v2"
)

func newPsCommand() *cli.Command {
	return &cli.Command{
		Name:  "ps",
		Usage: "List application status",
		Action: func(c *cli.Context) error {
			cfg, err := loadConfig(c)
			if err != nil {
				return err
			}

			tmuxClient, err := tmux.NewClient()
			if err != nil {
				return err
			}

			r := runner.New(cfg, tmuxClient)
			statuses, err := r.Status()
			if err != nil {
				return err
			}

			var rows []ui.TableRow
			for _, s := range statuses {
				pid := "-"
				if s.PID > 0 {
					pid = strconv.Itoa(s.PID)
				}
				rows = append(rows, ui.TableRow{
					Group:  s.Group,
					App:    s.App,
					Status: formatStatus(s),
					PID:    pid,
					Dir:    s.Dir,
				})
			}

			ui.PrintTable(os.Stdout, rows)
			return nil
		},
	}
}

// formatStatus renders e.g. "failed (1) 3s ago (5 restarts)".
func formatStatus(s runner.AppStatus) string {
	status := string(s.Status)
	if s.Exited {
		status = "exited"
		if s.RestartFailed {
			status = "failed"
		}
		status += " (" + formatExitReason(s) + ")"
		if !s.ExitedAt.IsZero() {
			status += " " + formatAgo(s.ExitedAt)
		}
	}
	if s.Restarts > 0 {
		status += fmt.Sprintf(" (%d %s)", s.Restarts, pluralize(s.Restarts, "restart"))
	}
	return status
}

func pluralize(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// formatExitReason describes why a pane died. tmux leaves the exit status
// empty for a pane killed by a signal, so the signal name is the only
// truthful thing to report there.
func formatExitReason(s runner.AppStatus) string {
	if s.ExitSignal != "" {
		return "SIG" + strings.ToUpper(s.ExitSignal)
	}
	return strconv.Itoa(s.ExitStatus)
}

func formatAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
