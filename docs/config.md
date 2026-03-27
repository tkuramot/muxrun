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
| `cmd` | string | Yes | Command to execute |
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

When `watch` is enabled, muxrun starts a background daemon that restarts the app on file changes. The daemon starts with `muxrun up` and stops with `muxrun down`.

- Logs: `$TMPDIR/muxrun/daemon-<group>.log`
- PID files: `$TMPDIR/muxrun/daemon-<group>.pid`

## Validation Rules

`muxrun check` validates the following:

- TOML syntax is correct
- At least one group exists
- Each group has a `name` and `dir`
- Each group has at least one app
- Each app has a `name` and `cmd`
- Group and app names contain only alphanumeric characters, hyphens, and underscores (`^[a-zA-Z0-9_-]+$`)
- No duplicate group names
- No duplicate app names within the same group
- All `exclude` patterns are valid regular expressions
