# muxrun

A CLI tool that manages and launches multiple applications in groups using tmux.

## Overview

muxrun manages multiple processes in a two-level hierarchy of groups and applications. Groups correspond to tmux sessions, and applications correspond to tmux windows. Commands defined in a config file can be started and stopped in bulk.

```
muxrun
├── Group A (= tmux session)
│   ├── App 1 (= tmux window)
│   └── App 2 (= tmux window)
└── Group B (= tmux session)
    └── App 3 (= tmux window)
```

## Use Cases

- **Microservice development** — Start multiple processes (API server, worker, frontend, etc.) at once with a single `muxrun up` instead of opening multiple terminals.
- **Auto-restart on file changes** — Enable `watch` to automatically restart apps when source files change, useful for languages without built-in hot reload (e.g., Go).
- **Selective group control** — Stop or restart only a subset of processes by group (e.g., `muxrun down backend`) without affecting the rest.

## Requirements

- tmux 3.0+
- fzf (only required for the `--interactive` option)

## Installation

```bash
go install github.com/tkuramot/muxrun@latest
```

## Configuration

Create a config file at `~/.config/muxrun/config.toml`.

```toml
[[group]]
name = "backend"

  [[group.app]]
  name = "api"
  cmd = "go run main.go"
  dir = "~/projects/myapp/cmd/api"
  watch = { enabled = true, exclude = ["_test\\.go$"] }

  [[group.app]]
  name = "worker"
  cmd = "go run worker.go"
  dir = "~/projects/myapp/cmd/worker"

[[group]]
name = "frontend"

  [[group.app]]
  name = "dev"
  cmd = "npm run dev"
  dir = "~/projects/frontend"
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `group.name` | string | Yes | Group name (used as tmux session name) |
| `group.app.name` | string | Yes | App name (used as tmux window name) |
| `group.app.cmd` | string | Yes | Command to execute |
| `group.app.dir` | string | Yes | Working directory |
| `group.app.watch` | object | No | File watch config (default: disabled) |

### Watch Configuration

```toml
# Enable watching
watch = { enabled = true }

# Enable watching with exclude patterns (regex)
watch = { enabled = true, exclude = ["_test\\.go$", "testdata/"] }
```

When file changes are detected, the application is automatically restarted. For groups with watch enabled, a background daemon is automatically started on `muxrun up` and stopped on `muxrun down`.

Daemon logs are stored at `$TMPDIR/muxrun/daemon-<group>.log` and PID files at `$TMPDIR/muxrun/daemon-<group>.pid`.

## Usage

### Validate config file

```bash
muxrun check
```

### Start applications

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
muxrun up -i
```

### Stop applications

```bash
# Stop all groups and apps
muxrun down

# Stop all apps in a specific group
muxrun down backend

# Stop multiple groups
muxrun down backend frontend

# Select interactively with fzf
muxrun down -i
```

### Check status

```bash
$ muxrun ps
GROUP       APP       STATUS    PID
backend     api       running   12345
backend     worker    running   12346
frontend    dev       stopped   -
```

## Working with tmux Sessions

Processes launched by muxrun run inside tmux sessions. Session names follow the format `muxrun-<group>`.

### Attaching to a session

```bash
# Attach to the backend group session
tmux attach -t muxrun-backend
```

Once attached, you can use standard tmux operations.

### Basic tmux operations (default prefix key: `Ctrl-b`)

| Action | Key | Description |
|--------|-----|-------------|
| Next window | `Ctrl-b` `n` | Switch to the next app (window) |
| Previous window | `Ctrl-b` `p` | Switch to the previous app (window) |
| Window list | `Ctrl-b` `w` | List all windows and select |
| Detach | `Ctrl-b` `d` | Detach from session (processes keep running) |
| Scroll mode | `Ctrl-b` `[` | Scroll through logs (`q` to exit) |

### Listing sessions and windows

```bash
# List muxrun-managed sessions
tmux list-sessions | grep muxrun-

# List windows (apps) in a specific session
tmux list-windows -t muxrun-backend
```

> [!WARNING]
> Use `muxrun down` to stop sessions and windows. Killing sessions directly with `tmux kill-session` may leave file watch daemons running.

> [!NOTE]
> If you manually stop a process inside an attached window, `muxrun ps` will reflect the updated status.

## Shell Completion

muxrun supports tab completion for zsh. Subcommands, flags, and group names are completed dynamically.

```bash
eval "$(muxrun completion zsh)"
```

To make it persistent, add the line to `~/.zshrc`.

## Development

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# E2E tests
go test -tags=e2e ./...
```

## License

MIT
