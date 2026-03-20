package cmd

import (
	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/runner"
	"github.com/tkuramot/muxrun/internal/selector"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/urfave/cli/v2"
)

func newUpCommand() *cli.Command {
	return &cli.Command{
		Name:  "up",
		Usage: "Start applications",
		ArgsUsage: "[group...]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "dir",
				Usage: "Override execution directory",
			},
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
				return upInteractive(c, cfg, r)
			}

			args := c.Args().Slice()
			if len(args) == 0 {
				return r.Up(c.Context, runner.UpOptions{
					DirOverride: c.String("dir"),
				})
			}
			for _, group := range args {
				if err := r.Up(c.Context, runner.UpOptions{
					GroupName:   group,
					DirOverride: c.String("dir"),
				}); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func upInteractive(c *cli.Context, cfg *config.Config, r *runner.Runner) error {
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
		if err := r.Up(c.Context, runner.UpOptions{
			GroupName:   s.Group,
			AppName:     s.App,
			DirOverride: c.String("dir"),
		}); err != nil {
			return err
		}
	}
	return nil
}

func loadConfig() (*config.Config, error) {
	path, err := config.DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
