package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func newCompletionCommand() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "Output shell completion script",
		ArgsUsage: "<bash|zsh|fish>",
		Action: func(c *cli.Context) error {
			shell := c.Args().First()
			if shell == "" {
				return fmt.Errorf("shell argument required, supported: bash, zsh, fish")
			}
			switch shell {
			case "bash":
				fmt.Print(bashCompletionScript)
			case "zsh":
				fmt.Print(zshCompletionScript)
			case "fish":
				fmt.Print(fishCompletionScript)
			default:
				return fmt.Errorf("unsupported shell %q, supported: bash, zsh, fish", shell)
			}
			return nil
		},
	}
}

func completeGroupNames(c *cli.Context) {
	cfg, err := loadConfig()
	if err != nil {
		return
	}
	for _, g := range cfg.Groups {
		fmt.Println(g.Name)
	}
}

const bashCompletionScript = `#! /bin/bash

_muxrun_complete() {
  local cur opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  opts=$(${COMP_WORDS[0]} "${COMP_WORDS[@]:1:$COMP_CWORD}" --generate-bash-completion)
  COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
  return 0
}

complete -o default -F _muxrun_complete muxrun
`

const zshCompletionScript = `#compdef muxrun

_muxrun() {
  local -a opts
  opts=("${(@f)$(${words[1]} ${words[@]:1} --generate-bash-completion)}")
  compadd -a opts
}

compdef _muxrun muxrun
`

const fishCompletionScript = `complete -c muxrun -f -a '(muxrun (commandline -cop) --generate-bash-completion)'
`
