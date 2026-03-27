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
		BashComplete: completeGroupNames,
		Action: func(c *cli.Context) error {
			cfg, configPath, err := loadConfigWithPath(c)
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
				if err := r.Up(runner.UpOptions{}); err != nil {
					return err
				}
				return spawnDaemons(cfg, configPath, "")
			}
			for _, group := range args {
				if err := r.Up(runner.UpOptions{
					GroupName: group,
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
