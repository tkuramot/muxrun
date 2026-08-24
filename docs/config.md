# Configuration Reference

## Config File Resolution

muxrun looks for a config file in the following order:

1. `--config / -c` flag (explicit path, skips other lookup)
2. `muxrun.toml` in the current directory, then parent directories up to the filesystem root

## Config Structure

```toml
# Group definition (at least one required)
[[group]]
name = "backend"                # Group name (required)
dir = "~/projects/myapp"        # Working directory for all apps in the group (required)

  # Application definition (at least one per group required)
  [[group.app]]
  name = "api"                  # App name (required)
  cmd = "go run main.go"       # Command to execute (required)
  restart = "on-failure"        # Restart policy (default: "no")
  watch = { enabled = true, exclude = ["_test\\.go$", "mock_.*\\.go$"] }

  [[group.app]]
  name = "worker"
  cmd = "go run worker.go"
  watch = { enabled = true, exclude = ["testdata/"] }

[[group]]
name = "frontend"
dir = "~/projects/frontend"

  [[group.app]]
  name = "dev"
  cmd = "npm run dev"
```

## Relative Paths and Git Worktrees

When `dir` is a relative path, it is resolved relative to the `muxrun.toml` location (not the current working directory). Using relative paths makes the config portable across git worktrees — copy `muxrun.toml` into each worktree and it resolves correctly without modification.

```toml
[[group]]
name = "backend"
dir = "."         # resolved relative to muxrun.toml location

  [[group.app]]
  name = "api"
  cmd = "go run main.go"
```

## Field Definitions

### Group (`[[group]]`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Group name. Used as the tmux session name (`muxrun-{name}`) |
| `dir` | string | Yes | Working directory for all apps in the group |

### Application (`[[group.app]]`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | App name. Used as the tmux window name |
| `cmd` | string | Yes | Command to execute. Runs as the tmux pane's process via the tmux `default-shell` started as an interactive login shell, so shell startup files apply and `muxrun ps` reports the command's own PID |
| `restart` | string | No | Restart policy: `"no"` (default) or `"on-failure"` |
| `watch` | bool \| object | No | File watch config (default: `false`) |

### Watch Options

`watch` can be specified in the following formats:

- `watch = false` -- Disabled (default)
- `watch = { enabled = true }` -- Enabled with no exclude patterns
- `watch = { enabled = true, exclude = [...] }` -- Enabled with exclude patterns

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Enable/disable file watching |
| `exclude` | string[] | No | Regex patterns to exclude from watching (default: empty) |

## Restart Policy

`restart` decides whether the group's daemon brings an app back after it exits on its own.

- `restart = "no"` -- Never restart (default). The pane stays dead and `muxrun ps` reports the exit status.
- `restart = "on-failure"` -- Restart the app when it fails.

An app "fails" when it exits non-zero, or dies on a signal it did not get from you. A clean `exit 0` is not a failure, and neither are `SIGINT`, `SIGTERM`, `SIGHUP` and `SIGQUIT` — pressing Ctrl-C in the pane stops the app for good, the way it does outside muxrun. `SIGKILL` counts as a failure, since it is usually an OOM kill rather than a deliberate stop.

Restarts back off — 1s, 2s, 4s, ... up to 30s — and stop after 5 consecutive failures, at which point `muxrun ps` reports the app as `failed`. An app that stays up for 10 seconds is considered recovered and gets the full retry budget back. So does an app restarted by `watch`: editing a file gives a failing app another 5 attempts.

`muxrun up` starts a fresh app, which resets the restart count.

```toml
  [[group.app]]
  name = "api"
  cmd = "go run main.go"
  restart = "on-failure"
  watch = { enabled = true }
```

On tmux older than 3.4, muxrun cannot tell an app killed by a signal from a clean exit, and does not restart it.

## Exclude Patterns

- Specified as regular expressions (Go `regexp` package syntax)
- Multiple patterns can be specified as an array
- Matched against relative file paths (relative to the working directory)
- Files matching any pattern are excluded from watching

```toml
watch = { enabled = true, exclude = [
  "_test\\.go$",      # Exclude test files
  "mock_.*\\.go$",    # Exclude mock files
  "testdata/",        # Exclude testdata directory
  "\\.tmp$",          # Exclude .tmp files
] }
```

## Daemon Files

When `watch` is enabled or `restart` is set to `"on-failure"`, muxrun starts a background daemon for the group. It restarts apps on file changes and after failures. The daemon starts with `muxrun up` and stops with `muxrun down`.

- Logs: `$TMPDIR/muxrun/daemon-<group>.log`
- PID files: `$TMPDIR/muxrun/daemon-<group>.pid`

## Validation Rules

`muxrun check` validates the following:

- TOML syntax is correct
- At least one group exists
- Each group has a `name` and `dir`
- Each group has at least one app
- Each app has a `name` and `cmd`
- Each app's `restart` is `"no"` or `"on-failure"`
- Group and app names contain only alphanumeric characters, hyphens, and underscores (`^[a-zA-Z0-9_-]+$`)
- No duplicate group names
- No duplicate app names within the same group
- All `exclude` patterns are valid regular expressions
