package config

import (
	"errors"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg, err := Load("../../testdata/valid_config.toml")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidate_EmptyGroups(t *testing.T) {
	cfg := &Config{}
	err := Validate(cfg)
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("expected ErrConfigValidation, got %v", err)
	}
}

func TestValidate_DuplicateGroupName(t *testing.T) {
	cfg, err := Load("../../testdata/duplicate_group.toml")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	err = Validate(cfg)
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("expected ErrConfigValidation, got %v", err)
	}
}

func TestValidate_DuplicateAppName(t *testing.T) {
	cfg, err := Load("../../testdata/duplicate_app.toml")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	err = Validate(cfg)
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("expected ErrConfigValidation, got %v", err)
	}
}

func TestValidate_InvalidGroupName(t *testing.T) {
	cfg, err := Load("../../testdata/invalid_name.toml")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	err = Validate(cfg)
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("expected ErrConfigValidation, got %v", err)
	}
}

func TestValidate_MissingCmd(t *testing.T) {
	cfg, err := Load("../../testdata/missing_required.toml")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	err = Validate(cfg)
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("expected ErrConfigValidation, got %v", err)
	}
}

func TestValidate_EmptyApps(t *testing.T) {
	cfg := &Config{
		Groups: []Group{
			{Name: "test", Apps: nil},
		},
	}
	err := Validate(cfg)
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("expected ErrConfigValidation, got %v", err)
	}
}

func TestValidate_InvalidExcludePattern(t *testing.T) {
	cfg := &Config{
		Groups: []Group{
			{
				Name: "test",
				Apps: []App{
					{
						Name:  "app",
						Cmd:   "echo",
						Dir:   "/tmp",
						Watch: WatchConfig{Enabled: true, Exclude: []string{"[invalid"}},
					},
				},
			},
		},
	}
	err := Validate(cfg)
	if !errors.Is(err, ErrConfigValidation) {
		t.Errorf("expected ErrConfigValidation, got %v", err)
	}
}
