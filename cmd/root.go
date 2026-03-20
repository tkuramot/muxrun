package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func NewApp() *cli.App {
	return &cli.App{
		Name:  "muxrun",
		Usage: "Manage multiple applications with tmux",
		Commands: []*cli.Command{
			newCheckCommand(),
			newUpCommand(),
			newDownCommand(),
			newPsCommand(),
			newDaemonCommand(),
		},
		ExitErrHandler: func(cCtx *cli.Context, err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		},
	}
}
