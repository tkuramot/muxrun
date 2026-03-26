package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func newCompletionCommand() *cli.Command {
	return &cli.Command{
		Name:      "completion",
		Usage:     "Output shell completion script",
		ArgsUsage: "<zsh>",
		Action: func(c *cli.Context) error {
			shell := c.Args().First()
			switch shell {
			case "zsh":
				fmt.Print(zshCompletionScript)
			default:
				return fmt.Errorf("unsupported shell %q, supported: zsh", shell)
			}
			return nil
		},
	}
}

func completeGroupNames(c *cli.Context) {
	cfg, err := loadConfig(c)
	if err != nil {
		return
	}
	for _, g := range cfg.Groups {
		fmt.Println(g.Name)
	}
}


func completeLogsArgs(c *cli.Context) {
	cfg, err := loadConfig(c)
	if err != nil {
		return
	}
	args := c.Args().Slice()
	switch len(args) {
	case 0:
		for _, g := range cfg.Groups {
			fmt.Println(g.Name)
		}
	case 1:
		// If args[0] exactly matches a group name, the group is complete — show apps.
		// Otherwise the user is still typing the group name — show groups.
		for _, g := range cfg.Groups {
			if g.Name == args[0] {
				for _, a := range g.Apps {
					fmt.Println(a.Name)
				}
				return
			}
		}
		for _, g := range cfg.Groups {
			fmt.Println(g.Name)
		}
	default:
		for _, g := range cfg.Groups {
			if g.Name == args[0] {
				for _, a := range g.Apps {
					fmt.Println(a.Name)
				}
				return
			}
		}
	}
}

const zshCompletionScript = `#compdef muxrun

_muxrun() {
  # Complete file paths after --config or -c
  if [[ "${words[CURRENT-1]}" == "--config" || "${words[CURRENT-1]}" == "-c" ]]; then
    _files
    return
  fi

  local -a opts
  opts=("${(@f)$(${words[1]} ${words[@]:1} --generate-bash-completion)}")
  _describe 'command' opts
}

compdef _muxrun muxrun
`
