package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func newCheckCommand() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Validate the configuration file",
		Action: func(c *cli.Context) error {
			_, err := loadConfig(c)
			if err != nil {
				fmt.Println("config: test failed")
				return err
			}

			fmt.Println("config: syntax is ok")
			fmt.Println("config: test is successful")
			return nil
		},
	}
}
