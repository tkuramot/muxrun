package cmd

import (
	"github.com/tkuramot/muxrun/internal/daemon"
	"github.com/tkuramot/muxrun/internal/runner"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/urfave/cli/v2"
)

func newDownCommand() *cli.Command {
	return &cli.Command{
		Name:  "down",
		Usage: "Stop applications",
		ArgsUsage: "[group...]",
		Flags: []cli.Flag{},
		BashComplete: completeGroupNames,
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

			args := c.Args().Slice()
			if len(args) == 0 {
				if err := r.Down(runner.DownOptions{}); err != nil {
					return err
				}
				// Stop all daemons
				for _, g := range cfg.Groups {
					daemon.StopDaemon(g.Name)
				}
				return nil
			}
			for _, group := range args {
				if err := r.Down(runner.DownOptions{GroupName: group}); err != nil {
					return err
				}
				daemon.StopDaemon(group)
			}
			return nil
		},
	}
}
