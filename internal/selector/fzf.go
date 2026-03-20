package selector

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrFzfNotAvailable = errors.New("fzf is not available")
	ErrFzfCancelled    = errors.New("fzf selection cancelled")
)

type AppOption struct {
	Group string
	App   string
}

func (o AppOption) String() string {
	return o.Group + "/" + o.App
}

func ParseAppOption(s string) (AppOption, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return AppOption{}, fmt.Errorf("invalid selection: %s", s)
	}
	return AppOption{Group: parts[0], App: parts[1]}, nil
}

func SelectApps(options []AppOption) ([]AppOption, error) {
	fzfPath, err := exec.LookPath("fzf")
	if err != nil {
		return nil, ErrFzfNotAvailable
	}

	var input strings.Builder
	for _, o := range options {
		input.WriteString(o.String())
		input.WriteString("\n")
	}

	cmd := exec.Command(fzfPath, "--multi", "--prompt", "Select apps> ")
	cmd.Stdin = strings.NewReader(input.String())
	cmd.Stderr = os.Stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil, ErrFzfCancelled
		}
		return nil, ErrFzfCancelled
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return nil, ErrFzfCancelled
	}

	var selected []AppOption
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		opt, err := ParseAppOption(line)
		if err != nil {
			return nil, err
		}
		selected = append(selected, opt)
	}
	return selected, nil
}
