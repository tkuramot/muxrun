package cmd

import (
	"fmt"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/runner"
	"github.com/tkuramot/muxrun/internal/selector"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/urfave/cli/v2"
)

func newDownCommand() *cli.Command {
	return &cli.Command{
		Name:  "down",
		Usage: "Stop applications",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "group",
				Aliases: []string{"g"},
				Usage:   "Target group name",
			},
			&cli.StringFlag{
				Name:    "app",
				Aliases: []string{"a"},
				Usage:   "Target app name (requires --group)",
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Usage:   "Select apps interactively with fzf",
			},
		},
		Action: func(c *cli.Context) error {
			if c.String("app") != "" && c.String("group") == "" {
				return fmt.Errorf("--app requires --group")
			}

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

			return r.Down(c.Context, runner.DownOptions{
				GroupName: c.String("group"),
				AppName:   c.String("app"),
			})
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

	for _, s := range selected {
		if err := r.Down(c.Context, runner.DownOptions{
			GroupName: s.Group,
			AppName:   s.App,
		}); err != nil {
			return err
		}
	}
	return nil
}
