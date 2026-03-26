package cmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/urfave/cli/v2"
)

var version string

func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func NewApp() *cli.App {
	return &cli.App{
		Name:                 "muxrun",
		Usage:                "Manage multiple applications with tmux",
		Version:              resolveVersion(),
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
			newLogsCommand(),
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
