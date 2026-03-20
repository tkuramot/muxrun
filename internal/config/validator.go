package config

import (
	"fmt"
	"regexp"
)

var nameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func Validate(cfg *Config) error {
	if len(cfg.Groups) == 0 {
		return fmt.Errorf("%w: at least one group is required", ErrConfigValidation)
	}

	groupNames := make(map[string]bool)
	for _, g := range cfg.Groups {
		if g.Name == "" {
			return fmt.Errorf("%w: group name is required", ErrConfigValidation)
		}
		if !nameRegexp.MatchString(g.Name) {
			return fmt.Errorf("%w: group name %q must contain only alphanumeric characters, hyphens, and underscores", ErrConfigValidation, g.Name)
		}
		if groupNames[g.Name] {
			return fmt.Errorf("%w: duplicate group name %q", ErrConfigValidation, g.Name)
		}
		groupNames[g.Name] = true

		if len(g.Apps) == 0 {
			return fmt.Errorf("%w: group %q must have at least one app", ErrConfigValidation, g.Name)
		}

		appNames := make(map[string]bool)
		for _, a := range g.Apps {
			if a.Name == "" {
				return fmt.Errorf("%w: app name is required in group %q", ErrConfigValidation, g.Name)
			}
			if !nameRegexp.MatchString(a.Name) {
				return fmt.Errorf("%w: app name %q in group %q must contain only alphanumeric characters, hyphens, and underscores", ErrConfigValidation, a.Name, g.Name)
			}
			if appNames[a.Name] {
				return fmt.Errorf("%w: duplicate app name %q in group %q", ErrConfigValidation, a.Name, g.Name)
			}
			appNames[a.Name] = true

			if a.Cmd == "" {
				return fmt.Errorf("%w: cmd is required for app %q in group %q", ErrConfigValidation, a.Name, g.Name)
			}
			if a.Dir == "" {
				return fmt.Errorf("%w: dir is required for app %q in group %q", ErrConfigValidation, a.Name, g.Name)
			}

			for _, pattern := range a.Watch.Exclude {
				if _, err := regexp.Compile(pattern); err != nil {
					return fmt.Errorf("%w: invalid exclude pattern %q for app %q in group %q: %s", ErrConfigValidation, pattern, a.Name, g.Name, err)
				}
			}
		}
	}

	return nil
}
