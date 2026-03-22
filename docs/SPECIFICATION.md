# muxrun Specification

## Overview

muxrun is a CLI tool that manages and launches multiple applications in groups using tmux.

## Core Concepts

### Structure

```
muxrun
├── Group A (= tmux session)
│   ├── App 1 (= tmux window)
│   └── App 2 (= tmux window)
└── Group B (= tmux session)
    └── App 3 (= tmux window)
```

- **Group**: Corresponds to a tmux session. A unit for organizing related applications.
- **Application**: Corresponds to a tmux window. The actual process being executed.

### Directory Specification

Each application requires a `dir` field. This can be overridden with the CLI `--dir` option.

### Watch Feature

- Monitors files under the application's working directory
- Automatically restarts the application when file changes are detected
- Enabled per application with `watch = true`

---

## Config File

### Resolution Order

1. `--config / -c` flag (explicit path)
2. `muxrun.toml` in CWD, then parent directories up to root
3. `~/.config/muxrun/muxrun.toml` (global fallback)

### Structure

```toml
# Group definition (at least one required)
[[group]]
name = "backend"            # Group name (required, used as tmux session name)

  # Application definition (at least one per group required)
  [[group.app]]
  name = "api"              # App name (required, used as tmux window name)
  cmd = "go run main.go"    # Command to execute (required)
  dir = "~/projects/myapp/cmd/api"  # Working directory (required)
  watch = { enabled = true, exclude = ["_test\\.go$", "mock_.*\\.go$"] }

  [[group.app]]
  name = "worker"
  cmd = "go run worker.go"
  dir = "~/projects/myapp/cmd/worker"
  watch = { enabled = true, exclude = ["testdata/"] }

[[group]]
name = "frontend"

  [[group.app]]
  name = "dev"
  cmd = "npm run dev"
  dir = "~/projects/frontend"
```

### Field Definitions

#### Group (`[[group]]`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Group name. Used as the tmux session name |

#### Application (`[[group.app]]`)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | App name. Used as the tmux window name |
| `cmd` | string | Yes | Command to execute |
| `dir` | string | Yes | Working directory |
| `watch` | bool \| object | No | File watch config. Default: false |

#### Watch Options

`watch` can be specified in the following formats:

- `watch = false` — Disabled (default)
- `watch = { enabled = true }` — Enabled (no exclude patterns)
- `watch = { enabled = true, exclude = [...] }` — Enabled (with exclude patterns)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | Yes | Enable/disable file watching |
| `exclude` | string[] | No | File patterns to exclude (regex). Default: empty |

### Directory Resolution Order

Application working directory:

1. If `--dir <path>` is specified → use that value
2. Otherwise → use the app's `dir` from config

---

## CLI Commands

### `muxrun check`

Validates the config file syntax and required fields.

```bash
$ muxrun check
config: syntax is ok
config: test is successful

$ muxrun check
config: syntax error at line 15: unexpected character
config: test failed
```

**Validations:**
- TOML syntax correctness
- Required fields exist (group name, app name, app dir, cmd)
- At least one group exists
- Each group has at least one app
- No duplicate group or app names
- Group and app names follow naming conventions

### `muxrun ps`

Displays running application status.

```
$ muxrun ps
GROUP       APP       STATUS    PID
backend     api       running   12345
backend     worker    running   12346
frontend    dev       stopped   -
```

### `muxrun up`

Starts applications.

```bash
# Start all groups and apps
muxrun up

# Start all apps in a specific group
muxrun up backend

# Start multiple groups
muxrun up backend frontend

# Override working directory
muxrun up backend --dir ~/projects/other-app

# Select interactively with fzf
muxrun up --interactive
muxrun up -i

# Restart already running apps
muxrun up --force
muxrun up backend -f
```

**Options:**

| Option | Short | Description |
|--------|-------|-------------|
| `--dir <path>` | — | Explicitly set working directory (overrides config dir) |
| `--interactive` | `-i` | Select apps interactively with fzf |
| `--force` | `-f` | Restart already running apps (kill existing window before re-creating) |

**Positional arguments:** `[group...]` — Target group names (all groups if omitted)

**Behavior:**
- No arguments: start all groups and apps
- Group name specified: start all apps in that group
- Multiple group names: start all apps in each group
- `--dir` specified: override config dir
- `--interactive` specified: select targets with fzf (multi-select)
- `--force` specified: kill existing windows and restart apps
- App already running (without `--force`): **error**

### `muxrun down`

Stops applications.

```bash
# Stop all groups and apps
muxrun down

# Stop all apps in a specific group
muxrun down backend

# Stop multiple groups
muxrun down backend frontend

# Select interactively with fzf
muxrun down --interactive
muxrun down -i
```

**Options:**

| Option | Short | Description |
|--------|-------|-------------|
| `--interactive` | `-i` | Select apps interactively with fzf |

**Positional arguments:** `[group...]` — Target group names (all groups if omitted)

**Behavior:**
- No arguments: stop all groups and apps, terminate all sessions
- Group name specified: stop all apps in that group, terminate session
- Multiple group names: stop all apps in each group, terminate sessions
- `--interactive` specified: select targets with fzf (multi-select)
- App already stopped: ignored (exits successfully)

---

## tmux Session Management

### Naming Convention

- Session name: `muxrun-{group_name}`
- Window name: `{app_name}`

### Lifecycle

1. `muxrun up group`: Create session `muxrun-group` (if it doesn't exist)
2. Create a window for each app and execute its command
3. `muxrun down group`: Close all windows and terminate the session
4. `muxrun down group app`: Close only that window. If it's the last window, the session is also terminated

---

## Watch Feature Details

### Monitored Targets

- All files under the application's working directory
- Hidden files and directories are excluded (`.git`, `node_modules`, etc.)
- Files matching `exclude` regex patterns are excluded

### Exclude Patterns

- Specified as regular expressions (Go `regexp` package syntax)
- Multiple patterns can be specified as an array
- Matched against relative file paths (relative to the working directory)
- Files matching any pattern are excluded from monitoring

```toml
watch = { enabled = true, exclude = [
  "_test\\.go$",      # Exclude test files
  "mock_.*\\.go$",    # Exclude mock files
  "testdata/",        # Exclude testdata directory
  "\\.tmp$",          # Exclude .tmp files
] }
```

### Restart Flow

1. File change detected
2. Debounce processing (coalesce consecutive changes into one)
3. Send SIGTERM to the current process
4. If no response after a timeout, send SIGKILL
5. Re-execute the command

---

## Error Handling

| Situation | Behavior |
|-----------|----------|
| Config file does not exist | Exit with error |
| Config syntax error | Exit with error, display line number |
| Specified group does not exist | Exit with error |
| Specified app does not exist | Exit with error |
| `up` on a running app (without `--force`) | Exit with error |
| `up --force` on a running app | Kill and restart |
| `down` on a stopped app | Exit successfully (no-op) |
| fzf cancelled | Exit with error |
| fzf not available | Exit with error |
| tmux not available | Exit with error |

---

## Constraints

- At least one group is required
- Each group must have at least one application
- Group and app names may only contain alphanumeric characters, hyphens, and underscores
- Duplicate app names within the same group are not allowed
- Duplicate group names are not allowed
