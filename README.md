# muxrun

`/mʌks.rʌn/`

Managing multiple processes across terminals gets messy: scattered tabs, no clear picture of what's running where, and manual restarts every time you switch branches or worktrees.

muxrun keeps it simple — `muxrun up` to start everything, `muxrun ps` to see what's running and from which directory, `muxrun up` again to restart.

## Quick Start

1. **Install muxrun:**

Homebrew (macOS / Linux):

```bash
brew install --cask tkuramot/tap/muxrun
```

Or via `go install`:

```bash
go install github.com/tkuramot/muxrun@latest
```

2. **Create `muxrun.toml` in your project directory:**

```toml
[[group]]
name = "myapp"
dir = "."

  [[group.app]]
  name = "server"
  cmd = "go run main.go"
```

3. **Run:**

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

## Configuration

### Config file resolution

muxrun looks for a config file in the following order:

1. `--config / -c` flag (explicit path, skips other lookup)
2. `muxrun.toml` in the current directory, then parent directories up to the filesystem root

### Minimal example

```toml
[[group]]
name = "backend"
dir = "."

  [[group.app]]
  name = "api"
  cmd = "go run main.go"
```

### Full example with file watching

```toml
[[group]]
name = "backend"
dir = "."

  [[group.app]]
  name = "api"
  cmd = "go run main.go"
  watch = { enabled = true, exclude = ["_test\\.go$"] }

  [[group.app]]
  name = "worker"
  cmd = "go run worker.go"

[[group]]
name = "frontend"
dir = "./frontend"

  [[group.app]]
  name = "dev"
  cmd = "npm run dev"
```

### Fields

See [docs/config.md](docs/config.md) for the full field reference, watch configuration, and validation rules.

When `watch` is enabled, muxrun starts a background daemon that restarts the app on file changes. The daemon starts with `muxrun up` and stops with `muxrun down`.

## Usage

### `muxrun up` — Start applications

```bash
muxrun up                           # Start all groups
muxrun up backend                   # Start a specific group
muxrun up backend frontend          # Start multiple groups
```

### `muxrun down` — Stop applications

`down` accepts the same arguments as `up`.

```bash
muxrun down                         # Stop all groups
muxrun down backend                 # Stop a specific group
```

> [!WARNING]
> Use `muxrun down` to stop sessions. Killing sessions directly with `tmux kill-session` may leave file watch daemons running.

### `muxrun ps` — Check status

```bash
$ muxrun ps
GROUP       APP       STATUS    PID      DIR
backend     api       running   12345    /home/user/repo/cmd/api
backend     worker    running   12346    /home/user/repo/cmd/worker
frontend    dev       stopped   -        /home/user/repo/frontend
```

### `muxrun logs` — View pane output

```bash
muxrun logs backend api       # Show buffered output for an app
muxrun logs -f backend api    # Stream output in real-time (Ctrl-C to stop)
```

> [!TIP]
> If you're familiar with tmux, each app runs in its own window inside a `muxrun-<group>` session — attach directly for full scrollback and search.

### `muxrun check` — Validate config file

```bash
muxrun check
```

## Tips

### Using with git worktrees

When `dir` is a relative path, it is resolved relative to the `muxrun.toml` location. Copy `muxrun.toml` into each worktree at creation time to treat each worktree as an independent environment.

```toml
[[group]]
name = "backend"
dir = "."          # resolved relative to muxrun.toml location

  [[group.app]]
  name = "api"
  cmd = "go run main.go"
```

With the same group name across worktrees, running `muxrun up` always restarts the apps for the current worktree — even if the session is already running from a different one:

```bash
# working in worktree-A
cd ~/repo-worktree-A && muxrun up   # muxrun-backend starts in A

# switch to worktree-B
cd ~/repo-worktree-B && muxrun up   # same session restarts in B
```

Copying `muxrun.toml` on worktree creation can be automated with a `post-checkout` hook or a worktree management tool like [git-worktree-runner](https://github.com/coderabbitai/git-worktree-runner).

## Claude Code Plugin

If you use [Claude Code](https://docs.anthropic.com/en/docs/claude-code), you can install the muxrun plugin to author and operate `muxrun.toml` with skills.

**Install**

```bash
claude plugin marketplace add https://github.com/tkuramot/muxrun
claude plugin install muxrun@muxrun
```

**Skills**

| Skill | Description |
|-------|-------------|
| `/muxrun-init` | Analyze your project structure and generate a tailored `muxrun.toml` |
| `/muxrun-import-compose` | Convert an existing `docker-compose.yml` / `Procfile` into `muxrun.toml` |
| `/muxrun-doctor` | Diagnose tmux/daemon/PID/config issues in the local environment |
| `/muxrun-triage-failure` | Find failing apps via `muxrun ps`, fetch logs, and pinpoint the cause |
| `/muxrun-tail` | Follow an app's logs and surface anomalies (panics, 5xx, restart loops) |
| `/muxrun-perf` | Diagnose slow startup / restart storms / over-broad watch and tune `exclude` |
| `/muxrun-bisect` | Drive `git bisect` for runtime regressions reproducible only with the dev stack up |

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
