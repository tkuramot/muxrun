package config

import (
	"errors"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	cfg, err := Load("../../testdata/valid_config.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(cfg.Groups))
	}

	g := cfg.Groups[0]
	if g.Name != "backend" {
		t.Errorf("expected group name 'backend', got %q", g.Name)
	}
	if len(g.Apps) != 2 {
		t.Fatalf("expected 2 apps in backend, got %d", len(g.Apps))
	}

	app := g.Apps[0]
	if app.Name != "api" {
		t.Errorf("expected app name 'api', got %q", app.Name)
	}
	if app.Cmd != "go run main.go" {
		t.Errorf("expected cmd 'go run main.go', got %q", app.Cmd)
	}
	if !app.Watch.Enabled {
		t.Error("expected watch.enabled to be true")
	}
	if len(app.Watch.Exclude) != 2 {
		t.Errorf("expected 2 exclude patterns, got %d", len(app.Watch.Exclude))
	}

	// Frontend group should have no watch
	fg := cfg.Groups[1]
	if fg.Apps[0].Watch.Enabled {
		t.Error("expected watch to be disabled for frontend/dev")
	}
}

func TestLoad_WatchOmitted(t *testing.T) {
	cfg, err := Load("../../testdata/watch_bool.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Groups[0].Apps[0].Watch.Enabled {
		t.Error("expected watch to be disabled when omitted for app1")
	}
	if cfg.Groups[0].Apps[1].Watch.Enabled {
		t.Error("expected watch to be disabled when omitted for app2")
	}
}

func TestLoad_InvalidSyntax(t *testing.T) {
	_, err := Load("../../testdata/invalid_syntax.toml")
	if err == nil {
		t.Fatal("expected error for invalid syntax")
	}
	if !errors.Is(err, ErrConfigSyntax) {
		t.Errorf("expected ErrConfigSyntax, got %v", err)
	}
}

func TestLoad_NotFound(t *testing.T) {
	_, err := Load("../../testdata/nonexistent.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestConfig_FindGroups(t *testing.T) {
	cfg := &Config{
		Groups: []Group{
			{Name: "backend"},
			{Name: "frontend"},
		},
	}

	groups := cfg.FindGroups("backend")
	if len(groups) != 1 || groups[0].Name != "backend" {
		t.Errorf("expected to find backend group")
	}

	groups = cfg.FindGroups("")
	if len(groups) != 2 {
		t.Errorf("expected all groups when name is empty")
	}

	groups = cfg.FindGroups("nonexistent")
	if len(groups) != 0 {
		t.Errorf("expected no groups for nonexistent name")
	}
}

func TestGroup_FindApp(t *testing.T) {
	g := &Group{
		Name: "test",
		Apps: []App{
			{Name: "api"},
			{Name: "worker"},
		},
	}

	app := g.FindApp("api")
	if app == nil || app.Name != "api" {
		t.Error("expected to find api app")
	}

	app = g.FindApp("nonexistent")
	if app != nil {
		t.Error("expected nil for nonexistent app")
	}
}
