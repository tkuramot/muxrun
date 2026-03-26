# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is muxrun

A Go CLI tool that launches and manages multiple applications in groups using tmux. Each group maps to a tmux session, each app to a tmux window. Supports file watching with auto-restart via background daemons.

## Build & Test

```bash
go build -o muxrun .           # Build
go test ./...                   # Unit tests
go test -tags=integration ./... # Integration tests (requires tmux)
go test -tags=e2e ./...         # E2E tests (requires tmux)
go test ./internal/config/...   # Run tests for a specific package
```

Dev environment uses Nix flakes (`nix develop` or direnv).

## Release

1. Update docs if affected by the changes
2. Update `version` in `flake.nix` to the new version
3. Commit the version bump
4. Create a git tag (`git tag vX.Y.Z`)
5. Push commits and tag (`git push origin main --tags`)

## Documentation

- **`docs/config.md`** — Authoritative source for configuration schema, field definitions, and validation rules. Referenced by the Claude Code plugin via symlink.
- **`docs/ARCHITECTURE.md`** — Technical architecture, daemon design, and dependency structure.

## Plugin

- **`plugin/.claude-plugin/plugin.json`** — Plugin metadata (name, version).
- **`plugin/skills/create-config/SKILL.md`** — Skill definition for `/create-config`.
- **`plugin/skills/create-config/config.md`** — Symlink to `docs/config.md`; updating docs automatically updates the skill's reference.

### Updating the plugin

1. Edit `plugin/skills/*/SKILL.md` for skill changes.
2. `config.md` is a symlink to `docs/config.md` — update the docs side and it auto-reflects.
3. Bump `version` in `plugin/.claude-plugin/plugin.json` for behavioral changes.
4. Update the plugin section in `README.md` if user-facing instructions change.

## Architecture

- **`cmd/`** — CLI commands using `urfave/cli/v2`. Each subcommand (`up`, `down`, `ps`, `logs`, `check`, `completion`) is its own file. The hidden `_daemon` command is the file-watcher daemon entry point.
- **`internal/runner/`** — Core orchestration logic. `Runner` takes a `config.Config` and a `tmux.Client` interface, coordinates starting/stopping apps and querying status.
- **`internal/tmux/`** — `Client` interface wrapping tmux shell commands. Has a mock implementation (`mock.go`) for unit testing.
- **`internal/config/`** — TOML config loading and validation. Config resolution: `--config` flag → `muxrun.toml` walking up from CWD. User-level CLI flag defaults in `~/.config/muxrun/config.toml`. Uses raw types for unmarshaling then converts to domain types.
- **`internal/daemon/`** — File-watch daemon lifecycle. `Spawn()` forks a detached `_daemon` process; `Run()` is the daemon main loop. PID files stored in `$TMPDIR/muxrun/`.
- **`internal/watcher/`** — File system watcher (fsnotify) with exclude filters and debouncing.
- **`internal/ui/`** — Table formatting for `ps` output.

## Key Design Patterns

- `tmux.Client` is an interface — all tmux interactions go through it, enabling unit tests via `tmux.MockClient`.
- tmux sessions are named `muxrun-<group>` (see `tmux.SessionName()`).
- Config uses a raw→domain conversion pattern: TOML tags on `rawConfig` structs, then converted to clean `Config`/`Group`/`App` types.
- Test fixtures live in `testdata/` as `.toml` files.
