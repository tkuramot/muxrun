package cmd

import (
	"github.com/tkuramot/muxrun/internal/daemon"
	"github.com/urfave/cli/v2"
)

func newDaemonCommand() *cli.Command {
	return &cli.Command{
		Name:   "_daemon",
		Hidden: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Required: true},
			&cli.StringFlag{Name: "group", Required: true},
		},
		Action: func(c *cli.Context) error {
			return daemon.Run(c.String("config"), c.String("group"))
		},
	}
}
