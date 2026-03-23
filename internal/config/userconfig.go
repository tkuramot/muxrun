package config

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

type UserConfig struct {
	Flags UserFlags
}

type UserFlags struct {
	Up UpFlags
}

type UpFlags struct {
	Force bool
}

// raw types for TOML unmarshaling
type rawUserConfig struct {
	Flags rawUserFlags `toml:"flags"`
}

type rawUserFlags struct {
	Up rawUpFlags `toml:"up"`
}

type rawUpFlags struct {
	Force bool `toml:"force"`
}

// LoadUserConfig reads ~/.config/muxrun/config.toml.
// If the file does not exist, it returns a zero-value UserConfig.
func LoadUserConfig() (*UserConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	path := filepath.Join(home, ".config", "muxrun", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &UserConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read user config: %w", err)
	}

	var raw rawUserConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse user config: %w", err)
	}

	return &UserConfig{
		Flags: UserFlags{
			Up: UpFlags{
				Force: raw.Flags.Up.Force,
			},
		},
	}, nil
}
