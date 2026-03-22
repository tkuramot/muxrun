package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Groups []Group
}

type Group struct {
	Name string
	Dir  string
	Apps []App
}

type App struct {
	Name  string
	Cmd   string
	Watch WatchConfig
}

type WatchConfig struct {
	Enabled bool
	Exclude []string
}

// raw types for TOML unmarshaling
type rawConfig struct {
	Groups []rawGroup `toml:"group"`
}

type rawGroup struct {
	Name string   `toml:"name"`
	Dir  string   `toml:"dir"`
	Apps []rawApp `toml:"app"`
}

type rawWatchConfig struct {
	Enabled bool     `toml:"enabled"`
	Exclude []string `toml:"exclude"`
}

type rawApp struct {
	Name  string         `toml:"name"`
	Cmd   string         `toml:"cmd"`
	Watch rawWatchConfig `toml:"watch"`
}

const configFileName = "muxrun.toml"

// ResolveConfigPath determines which config file to use.
// Priority: explicit path > muxrun.toml in CWD or ancestor > ~/.config/muxrun/muxrun.toml
func ResolveConfigPath(explicit string) (string, error) {
	if explicit != "" {
		return expandPath(explicit)
	}

	// Walk from CWD to root looking for muxrun.toml
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	for {
		candidate := filepath.Join(dir, configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback to global config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "muxrun", configFileName), nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var raw rawConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		var derr *toml.DecodeError
		if ok := isDecodeError(err, &derr); ok {
			row, col := derr.Position()
			return nil, &ConfigSyntaxError{
				Line:    row,
				Column:  col,
				Message: derr.Error(),
			}
		}
		return nil, fmt.Errorf("%w: %s", ErrConfigSyntax, err)
	}

	cfg, err := convertRawConfig(&raw)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func convertRawConfig(raw *rawConfig) (*Config, error) {
	cfg := &Config{}
	for _, rg := range raw.Groups {
		dir, err := expandPath(rg.Dir)
		if err != nil {
			return nil, err
		}
		g := Group{Name: rg.Name, Dir: dir}
		for _, ra := range rg.Apps {
			g.Apps = append(g.Apps, App{
				Name: ra.Name,
				Cmd:  ra.Cmd,
				Watch: WatchConfig{
					Enabled: ra.Watch.Enabled,
					Exclude: ra.Watch.Exclude,
				},
			})
		}
		cfg.Groups = append(cfg.Groups, g)
	}
	return cfg, nil
}

func isDecodeError(err error, target **toml.DecodeError) bool {
	if de, ok := err.(*toml.DecodeError); ok {
		*target = de
		return true
	}
	return false
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand ~: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// FindGroups returns groups matching the given name, or all groups if name is empty.
func (c *Config) FindGroups(name string) []Group {
	if name == "" {
		return c.Groups
	}
	for _, g := range c.Groups {
		if g.Name == name {
			return []Group{g}
		}
	}
	return nil
}

// FindApp returns the app with the given name in the group.
func (g *Group) FindApp(name string) *App {
	for i := range g.Apps {
		if g.Apps[i].Name == name {
			return &g.Apps[i]
		}
	}
	return nil
}
