package cmd

import (
	"fmt"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/tkuramot/muxrun/internal/daemon"
	"github.com/tkuramot/muxrun/internal/runner"
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
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Restart already running apps",
			},
		},
		BashComplete: completeGroupNames,
		Action: func(c *cli.Context) error {
			cfg, configPath, err := loadConfigWithPath(c)
			if err != nil {
				return err
			}

			userCfg, err := config.LoadUserConfig()
			if err != nil {
				return err
			}

			force := c.Bool("force")
			if !c.IsSet("force") {
				force = userCfg.Flags.Up.Force
			}

			tmuxClient, err := tmux.NewClient()
			if err != nil {
				return err
			}

			r := runner.New(cfg, tmuxClient)

			args := c.Args().Slice()
			if len(args) == 0 {
				if err := r.Up(runner.UpOptions{
					DirOverride: c.String("dir"),
					Force:       force,
				}); err != nil {
					return err
				}
				return spawnDaemons(cfg, configPath, "")
			}
			for _, group := range args {
				if err := r.Up(runner.UpOptions{
					GroupName:   group,
					DirOverride: c.String("dir"),
					Force:       force,
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

func loadConfig(c *cli.Context) (*config.Config, error) {
	cfg, _, err := loadConfigWithPath(c)
	return cfg, err
}

func loadConfigWithPath(c *cli.Context) (*config.Config, string, error) {
	path, err := config.ResolveConfigPath(c.String("config"))
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
