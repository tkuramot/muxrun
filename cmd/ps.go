package cmd

import (
	"os"
	"strconv"

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
			cfg, err := loadConfig()
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
					Status: string(s.Status),
					PID:    pid,
					Dir:    s.Dir,
				})
			}

			ui.PrintTable(os.Stdout, rows)
			return nil
		},
	}
}
