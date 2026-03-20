package cmd

import (
	"fmt"

	"github.com/tkuramot/muxrun/internal/config"
	"github.com/urfave/cli/v2"
)

func newCheckCommand() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Validate the configuration file",
		Action: func(c *cli.Context) error {
			path, err := config.DefaultConfigPath()
			if err != nil {
				return err
			}

			cfg, err := config.Load(path)
			if err != nil {
				fmt.Println("config: test failed")
				return err
			}

			if err := config.Validate(cfg); err != nil {
				fmt.Println("config: test failed")
				return err
			}

			fmt.Println("config: syntax is ok")
			fmt.Println("config: test is successful")
			return nil
		},
	}
}
