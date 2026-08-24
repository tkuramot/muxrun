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
				statusStr := string(s.Status)
				if s.Exited {
					statusStr = "exited (" + formatExitReason(s) + ")"
					if !s.ExitedAt.IsZero() {
						statusStr += " " + formatAgo(s.ExitedAt)
					}
				}
				rows = append(rows, ui.TableRow{
					Group:  s.Group,
					App:    s.App,
					Status: statusStr,
					PID:    pid,
					Dir:    s.Dir,
				})
			}

			ui.PrintTable(os.Stdout, rows)
			return nil
		},
	}
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
