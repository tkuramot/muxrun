package cmd

import (
	"fmt"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/daemon"
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
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Restart already running apps",
			},
		},
		BashComplete: completeGroupNames,
		Action: func(c *cli.Context) error {
			cfg, configPath, err := loadConfigWithPath()
			if err != nil {
				return err
			}

			tmuxClient, err := tmux.NewClient()
			if err != nil {
				return err
			}

			r := runner.New(cfg, tmuxClient)

			if c.Bool("interactive") {
				return upInteractive(c, cfg, r, configPath)
			}

			args := c.Args().Slice()
			if len(args) == 0 {
				if err := r.Up(c.Context, runner.UpOptions{
					DirOverride: c.String("dir"),
					Force:       c.Bool("force"),
				}); err != nil {
					return err
				}
				return spawnDaemons(cfg, configPath, "")
			}
			for _, group := range args {
				if err := r.Up(c.Context, runner.UpOptions{
					GroupName:   group,
					DirOverride: c.String("dir"),
					Force:       c.Bool("force"),
				}); err != nil {
					return err
				}
				if err := spawnDaemons(cfg, configPath, group); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func upInteractive(c *cli.Context, cfg *config.Config, r *runner.Runner, configPath string) error {
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

	groups := make(map[string]bool)
	for _, s := range selected {
		if err := r.Up(c.Context, runner.UpOptions{
			GroupName:   s.Group,
			AppName:     s.App,
			DirOverride: c.String("dir"),
			Force:       c.Bool("force"),
		}); err != nil {
			return err
		}
		groups[s.Group] = true
	}

	for group := range groups {
		if err := spawnDaemons(cfg, configPath, group); err != nil {
			return err
		}
	}
	return nil
}

func spawnDaemons(cfg *config.Config, configPath, groupName string) error {
	groups := cfg.FindGroups(groupName)
	for _, g := range groups {
		hasWatch := false
		for _, app := range g.Apps {
			if app.Watch.Enabled {
				hasWatch = true
				break
			}
		}
		if !hasWatch {
			continue
		}

		// Stop existing daemon and respawn
		if daemon.IsRunning(g.Name) {
			daemon.StopDaemon(g.Name)
		}
		if err := daemon.Spawn(configPath, g.Name); err != nil {
			fmt.Printf("warning: failed to start daemon for group %s: %v\n", g.Name, err)
			continue
		}
		fmt.Printf("started file watcher daemon for group %s\n", g.Name)
	}
	return nil
}

func loadConfig() (*config.Config, error) {
	cfg, _, err := loadConfigWithPath()
	return cfg, err
}

func loadConfigWithPath() (*config.Config, string, error) {
	path, err := config.DefaultConfigPath()
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	if err := config.Validate(cfg); err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}
