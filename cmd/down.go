package cmd

import (
	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/daemon"
	"github.com/tkuramot/muxrun/internal/runner"
	"github.com/tkuramot/muxrun/internal/selector"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/urfave/cli/v2"
)

func newDownCommand() *cli.Command {
	return &cli.Command{
		Name:  "down",
		Usage: "Stop applications",
		ArgsUsage: "[group...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Select apps interactively with fzf",
			},
		},
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

			if c.Bool("interactive") {
				return downInteractive(c, cfg, r)
			}

			args := c.Args().Slice()
			if len(args) == 0 {
				if err := r.Down(c.Context, runner.DownOptions{}); err != nil {
					return err
				}
				// Stop all daemons
				for _, g := range cfg.Groups {
					daemon.StopDaemon(g.Name)
				}
				return nil
			}
			for _, group := range args {
				if err := r.Down(c.Context, runner.DownOptions{GroupName: group}); err != nil {
					return err
				}
				daemon.StopDaemon(group)
			}
			return nil
		},
	}
}

func downInteractive(c *cli.Context, cfg *config.Config, r *runner.Runner) error {
	var options []selector.AppOption
	for _, g := range cfg.Groups {
		for _, a := range g.Apps {
			options = append(options, selector.AppOption{Group: g.Name, App: a.Name})
		}
	}

	selected, err := selector.SelectApps(options)
	if err != nil {
		return err
	}

	// Track which groups had all apps stopped
	groupApps := make(map[string]int)
	for _, g := range cfg.Groups {
		groupApps[g.Name] = len(g.Apps)
	}
	stoppedPerGroup := make(map[string]int)

	for _, s := range selected {
		if err := r.Down(c.Context, runner.DownOptions{
			GroupName: s.Group,
			AppName:   s.App,
		}); err != nil {
			return err
		}
		stoppedPerGroup[s.Group]++
	}

	// Stop daemon for groups where all apps were stopped
	for group, stopped := range stoppedPerGroup {
		if stopped >= groupApps[group] {
			daemon.StopDaemon(group)
		}
	}
	return nil
}
