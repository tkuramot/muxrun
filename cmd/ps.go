package cmd

import (
	"fmt"
	"os"
	"strconv"
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
					statusStr = fmt.Sprintf("exited (%d)", s.ExitStatus)
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
