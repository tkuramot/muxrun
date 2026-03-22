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
