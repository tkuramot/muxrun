package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUserConfig_FileNotExist(t *testing.T) {
	// Set HOME to a temp dir with no config.toml
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	uc, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uc.Flags.Up.Force {
		t.Error("expected Force to be false when file does not exist")
	}
}

func TestLoadUserConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	configDir := filepath.Join(dir, ".config", "muxrun")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `[flags.up]
force = true
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	uc, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !uc.Flags.Up.Force {
		t.Error("expected Force to be true")
	}
}
