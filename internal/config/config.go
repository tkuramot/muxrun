package config

import (
	"errors"
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

type rawApp struct {
	Name string `toml:"name"`
	Cmd  string `toml:"cmd"`
	// watch is documented in both a shorthand and a table form, so it is
	// decoded loosely and converted by parseWatch.
	Watch any `toml:"watch"`
}

const configFileName = "muxrun.toml"

// ResolveConfigPath determines which config file to use.
// Priority: explicit path > muxrun.toml in CWD or ancestor.
// Returns ErrConfigNotFound if no config file is found.
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

	return "", ErrConfigNotFound
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
		if errors.As(err, &derr) {
			row, col := derr.Position()
			return nil, &ConfigSyntaxError{
				Line:    row,
				Column:  col,
				Message: derr.Error(),
			}
		}
		return nil, fmt.Errorf("%w: %s", ErrConfigSyntax, err)
	}

	configDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config directory: %w", err)
	}

	cfg, err := convertRawConfig(&raw, configDir)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func convertRawConfig(raw *rawConfig, configDir string) (*Config, error) {
	cfg := &Config{}
	for _, rg := range raw.Groups {
		dir, err := expandPath(rg.Dir)
		if err != nil {
			return nil, err
		}
		if dir != "" && !filepath.IsAbs(dir) {
			dir = filepath.Join(configDir, dir)
		}
		g := Group{Name: rg.Name, Dir: dir}
		for _, ra := range rg.Apps {
			watch, err := parseWatch(ra.Watch, rg.Name, ra.Name)
			if err != nil {
				return nil, err
			}
			g.Apps = append(g.Apps, App{
				Name:  ra.Name,
				Cmd:   ra.Cmd,
				Watch: watch,
			})
		}
		cfg.Groups = append(cfg.Groups, g)
	}
	return cfg, nil
}

// parseWatch converts the watch field, which may be either the shorthand
// `watch = false` or a table `watch = { enabled = true, exclude = [...] }`.
func parseWatch(v any, group, app string) (WatchConfig, error) {
	switch w := v.(type) {
	case nil:
		return WatchConfig{}, nil
	case bool:
		return WatchConfig{Enabled: w}, nil
	case map[string]any:
		cfg := WatchConfig{}
		if enabled, ok := w["enabled"]; ok {
			b, ok := enabled.(bool)
			if !ok {
				return WatchConfig{}, watchFieldError(group, app, "watch.enabled must be a boolean")
			}
			cfg.Enabled = b
		}
		if exclude, ok := w["exclude"]; ok {
			items, ok := exclude.([]any)
			if !ok {
				return WatchConfig{}, watchFieldError(group, app, "watch.exclude must be an array of strings")
			}
			for _, item := range items {
				pattern, ok := item.(string)
				if !ok {
					return WatchConfig{}, watchFieldError(group, app, "watch.exclude must be an array of strings")
				}
				cfg.Exclude = append(cfg.Exclude, pattern)
			}
		}
		return cfg, nil
	default:
		return WatchConfig{}, watchFieldError(group, app, "watch must be a boolean or a table")
	}
}

func watchFieldError(group, app, msg string) error {
	return fmt.Errorf("%w: %s for app %q in group %q", ErrConfigValidation, msg, app, group)
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
