# muxrun

A CLI tool that launches and manages multiple applications in groups using tmux.

## Quick Start

1. Install muxrun:

```bash
go install github.com/tkuramot/muxrun@latest
```

2. Create `~/.config/muxrun/config.toml`:

```toml
[[group]]
name = "myapp"

  [[group.app]]
  name = "server"
  cmd = "go run main.go"
  dir = "~/projects/myapp"
```

3. Run:

```bash
muxrun up
```

This starts the `server` app inside a tmux session named `muxrun-myapp`. Use `muxrun down` to stop it.

## Overview

muxrun organizes applications into **groups**. Each group becomes a tmux session, and each app becomes a window within that session.

```
muxrun
├── Group A (tmux session)
│   ├── App 1 (tmux window)
│   └── App 2 (tmux window)
└── Group B (tmux session)
    └── App 3 (tmux window)
```

## Requirements

- tmux 3.0+
- fzf (only for the `--interactive` option)

## Configuration

### Minimal example

```toml
[[group]]
name = "backend"

  [[group.app]]
  name = "api"
  cmd = "go run main.go"
  dir = "~/projects/myapp/cmd/api"
```

### Full example with file watching

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
| `name` (under `[[group]]`) | string | Yes | Group name (tmux session name) |
| `name` (under `[[group.app]]`) | string | Yes | App name (tmux window name) |
| `cmd` | string | Yes | Command to execute |
| `dir` | string | Yes | Working directory |
| `watch.enabled` | bool | No | Enable file watching (default: `false`) |
| `watch.exclude` | string[] | No | Regex patterns to exclude from watching |

When `watch` is enabled, muxrun starts a background daemon that restarts the app on file changes. The daemon starts with `muxrun up` and stops with `muxrun down`.

Daemon logs: `$TMPDIR/muxrun/daemon-<group>.log` / PID files: `$TMPDIR/muxrun/daemon-<group>.pid`

## Usage

### `muxrun up` — Start applications

```bash
muxrun up                           # Start all groups
muxrun up backend                   # Start a specific group
muxrun up backend frontend          # Start multiple groups
muxrun up backend --dir ~/other     # Override working directory
muxrun up -i                        # Select interactively with fzf
```

### `muxrun down` — Stop applications

`down` accepts the same arguments as `up`.

```bash
muxrun down                         # Stop all groups
muxrun down backend                 # Stop a specific group
muxrun down -i                      # Select interactively with fzf
```

### `muxrun ps` — Check status

```bash
$ muxrun ps
GROUP       APP       STATUS    PID      DIR
backend     api       running   12345    ~/projects/myapp/cmd/api
backend     worker    running   12346    ~/projects/myapp/cmd/worker
frontend    dev       stopped   -        ~/projects/frontend
```

### `muxrun check` — Validate config file

```bash
muxrun check
```

## Use Cases

- **Microservice development** — Start multiple processes (API server, worker, frontend, etc.) at once with `muxrun up` instead of opening multiple terminals.
- **Auto-restart on file changes** — Enable `watch` to restart apps when source files change, useful for languages without built-in hot reload (e.g., Go).
- **Selective group control** — Stop or restart only a subset of processes by group (e.g., `muxrun down backend`) without affecting the rest.

## Working with tmux Sessions

muxrun creates tmux sessions named `muxrun-<group>`. Attach to a session to see app output:

```bash
tmux attach -t muxrun-backend
```

> [!WARNING]
> Use `muxrun down` to stop sessions. Killing sessions directly with `tmux kill-session` may leave file watch daemons running.

> [!NOTE]
> If you manually stop a process inside an attached window, `muxrun ps` reflects the updated status.

<details>
<summary>Basic tmux operations (prefix: Ctrl-b)</summary>

| Action | Key | Description |
|--------|-----|-------------|
| Next window | `Ctrl-b` `n` | Switch to the next app |
| Previous window | `Ctrl-b` `p` | Switch to the previous app |
| Window list | `Ctrl-b` `w` | List all windows and select |
| Detach | `Ctrl-b` `d` | Detach from session (processes keep running) |
| Scroll mode | `Ctrl-b` `[` | Scroll through logs (`q` to exit) |

```bash
# List muxrun-managed sessions
tmux list-sessions | grep muxrun-

# List windows in a session
tmux list-windows -t muxrun-backend
```

</details>

## Shell Completion

muxrun supports tab completion for zsh. Subcommands, flags, and group names are completed dynamically.

```bash
eval "$(muxrun completion zsh)"
```

To make it persistent, add the line to `~/.zshrc`.

## Development

```bash
go test ./...                        # Unit tests
go test -tags=integration ./...      # Integration tests
go test -tags=e2e ./...              # E2E tests
```

## License

MIT
