package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tkuramot/muxrun/internal/runner"
	"github.com/tkuramot/muxrun/internal/tmux"
	"github.com/urfave/cli/v2"
)

func newLogsCommand() *cli.Command {
	return &cli.Command{
		Name:         "logs",
		Usage:        "Show pane output for a running application",
		ArgsUsage:    "<group> <app>",
		BashComplete: completeLogsArgs,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Stream output in real-time",
			},
		},
		Action: func(c *cli.Context) error {
			args := c.Args().Slice()
			if len(args) < 2 {
				return fmt.Errorf("group and app name are required")
			}
			if len(args) > 2 {
				return fmt.Errorf("too many arguments")
			}

			cfg, err := loadConfig(c)
			if err != nil {
				return err
			}

			tmuxClient, err := tmux.NewClient()
			if err != nil {
				return err
			}

			r := runner.New(cfg, tmuxClient)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigCh)
			go func() {
				<-sigCh
				cancel()
			}()

			err = r.Logs(ctx, runner.LogsOptions{
				GroupName: args[0],
				AppName:   args[1],
				Follow:    c.Bool("follow"),
				Writer:    os.Stdout,
			})
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		},
	}
}
