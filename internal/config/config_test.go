package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestResolveConfigPath_ExplicitPath(t *testing.T) {
	path, err := ResolveConfigPath("/tmp/custom.toml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/tmp/custom.toml" {
		t.Errorf("expected /tmp/custom.toml, got %q", path)
	}
}

func TestResolveConfigPath_CWD(t *testing.T) {
	dir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var)
	dir, _ = filepath.EvalSymlinks(dir)
	configFile := filepath.Join(dir, "muxrun.toml")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(dir)

	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != configFile {
		t.Errorf("expected %q, got %q", configFile, path)
	}
}

func TestResolveConfigPath_ParentDir(t *testing.T) {
	parent := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var)
	parent, _ = filepath.EvalSymlinks(parent)
	configFile := filepath.Join(parent, "muxrun.toml")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	child := filepath.Join(parent, "subdir")
	os.Mkdir(child, 0755)

	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(child)

	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != configFile {
		t.Errorf("expected %q, got %q", configFile, path)
	}
}

func TestResolveConfigPath_GlobalFallback(t *testing.T) {
	// Use a temp dir with no muxrun.toml so it falls back to global
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(dir)

	path, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(".config", "muxrun", "muxrun.toml")) {
		t.Errorf("expected global fallback path, got %q", path)
	}
}

func TestLoad_RelativeDirResolvedFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)

	configContent := `
[[group]]
name = "backend"
dir = "./backend"

[[group.app]]
name = "api"
cmd = "go run main.go"
`
	configFile := filepath.Join(dir, "muxrun.toml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDir := filepath.Join(dir, "backend")
	if cfg.Groups[0].Dir != expectedDir {
		t.Errorf("expected dir %q, got %q", expectedDir, cfg.Groups[0].Dir)
	}
}

func TestLoad_AbsoluteDirUnchanged(t *testing.T) {
	dir := t.TempDir()

	configContent := `
[[group]]
name = "backend"
dir = "/absolute/path"

[[group.app]]
name = "api"
cmd = "go run main.go"
`
	configFile := filepath.Join(dir, "muxrun.toml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Groups[0].Dir != "/absolute/path" {
		t.Errorf("expected dir '/absolute/path', got %q", cfg.Groups[0].Dir)
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
