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

Releases are driven by the `release-muxrun` GitHub Actions workflow (`.github/workflows/release-muxrun.yml`), which runs GoReleaser to build binaries and publish the Homebrew cask to `tkuramot/homebrew-tap`.

The plugin/skills are released by the `release-muxrun-skills` workflow (`.github/workflows/release-muxrun-skills.yml`). It triggers on pushes to `main` that change `plugin/.claude-plugin/plugin.json`: if the `version` field there points to a not-yet-released version, the workflow tags `muxrun-skills-v<version>` and publishes a GitHub release with a tarball of `plugin/`. To cut a skills release, bump `version` in `plugin/.claude-plugin/plugin.json` and commit to `main` — no tag push needed.

Trigger it from the Actions tab via `workflow_dispatch` with a `patch`/`minor`/`major` bump. The workflow runs `scripts/bump-version.sh`, pushes the version commit and tag, and GoReleaser takes over.

The `HOMEBREW_TAP_GITHUB_TOKEN` secret must be set on the repo for the Homebrew cask publish step to succeed.

## Documentation

- **`docs/config.md`** — Authoritative source for configuration schema, field definitions, and validation rules. Referenced by the Claude Code plugin via symlink.
- **`docs/ARCHITECTURE.md`** — Technical architecture, daemon design, and dependency structure.

## Plugin

- **`plugin/.claude-plugin/plugin.json`** — Plugin metadata (name, version).
- **`plugin/skills/<name>/SKILL.md`** — Skill definitions. Current skills: `muxrun-init`, `muxrun-import-compose`, `muxrun-doctor`, `muxrun-triage-failure`, `muxrun-tail`, `muxrun-perf`, `muxrun-bisect`.
- **`plugin/skills/muxrun-init/config.md`**, **`plugin/skills/muxrun-import-compose/config.md`** — Symlinks to `docs/config.md`; updating the docs auto-updates the skills' references.

### Updating the plugin

1. Edit `plugin/skills/*/SKILL.md` for skill changes.
2. The `config.md` symlinks track `docs/config.md` — edit the docs side.
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
