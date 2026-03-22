package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func NewApp() *cli.App {
	return &cli.App{
		Name:                 "muxrun",
		Usage:                "Manage multiple applications with tmux",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to config file",
			},
		},
		Commands: []*cli.Command{
			newCheckCommand(),
			newUpCommand(),
			newDownCommand(),
			newPsCommand(),
			newDaemonCommand(),
			newCompletionCommand(),
		},
		ExitErrHandler: func(cCtx *cli.Context, err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}
