# muxrun Architecture

## 1. Tech Stack

| Library | Purpose | Rationale |
|---------|---------|-----------|
| `pelletier/go-toml/v2` | TOML parsing | TOML 1.0 compliant, detailed error messages |
| `urfave/cli/v2` | CLI framework | Built-in subcommands, aliases, and help generation |
| `fsnotify/fsnotify` | File watching | De facto standard, cross-platform support |

---

## 2. Directory Structure

```
muxrun/
├── cmd/            # CLI command definitions (one file per subcommand)
├── internal/
│   ├── config/     # TOML config loading and validation
│   ├── tmux/       # tmux client interface and implementation
│   ├── watcher/    # File system watching, debouncing, exclude filters
│   ├── daemon/     # Watch/restart daemon spawning, supervision, PID management
│   ├── runner/     # App start/stop/status orchestration
│   └── ui/         # Table formatting for output
├── docs/           # Documentation
└── testdata/       # Test fixture TOML files
```

- **`internal/`**: Go's language-level import restriction prevents external use, allowing free refactoring.
- **Feature-based packages**: Each package has a single responsibility.

---

## 3. Layered Architecture

```
┌─────────────────────────────────────────────────┐
│                   cmd/ (CLI Layer)               │
│  - Command-line argument parsing                 │
│  - User input validation                         │
│  - Output formatting                             │
│  - Daemon spawn/stop control                     │
└─────────────────────┬───────────────────────────┘
                      │
                      ▼
┌──────────────────────────┐  ┌────────────────────────────┐
│ internal/runner/         │  │ internal/daemon/            │
│ (Application)            │  │ (Watch / Restart Daemon)    │
│ - App start/stop         │  │ - Daemon process spawning   │
│ - Concurrent app control │  │ - PID file management       │
└───────┬──────────────────┘  │ - File change → app restart │
        │                     │ - Failure → app restart     │
        │                     └──────┬─────────────────────┘
        │                            │
        ▼                            ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│ internal/tmux │  │internal/watcher│ │internal/config│
│ - Session mgmt│  │ - File watching│ │ - TOML parsing │
│ - Window mgmt │  │ - Debouncing   │ │ - Validation   │
└───────────────┘  └───────────────┘  └───────────────┘
```

### Dependency Direction

- `cmd/` → `internal/runner/`, `internal/daemon/`, `internal/config/`
- `internal/runner/` → `internal/tmux/`, `internal/config/`
- `internal/daemon/` → `internal/tmux/`, `internal/watcher/`, `internal/config/`, `internal/runner/`
- Each `internal/` package is loosely coupled (`daemon` only references `runner`'s `process.go` utilities)

---

## 4. Daemon Architecture

### Independent daemon process per group

`muxrun up` spawns a separate daemon process for each group that has an app with `watch` enabled or `restart = "on-failure"`.

```
muxrun up
  ├── daemon (group: frontend)   ← PID 1234
  └── daemon (group: backend)    ← PID 5678
```

**Design rationale:**

| Aspect | Per-group daemon (chosen) | Single daemon |
|--------|--------------------------|---------------|
| Lifecycle management | `up`/`down` is self-contained per group. Just kill → respawn | Requires config hot-reload and partial update logic |
| Fault isolation | One process crash doesn't affect other groups | All groups go down together |
| Implementation simplicity | `Spawn()` / `StopDaemon()` managed with a single PID file | Must manage dynamic addition/removal of groups within a single process |

### Spawn mechanism

`Spawn()` re-executes its own binary with the hidden `_daemon` subcommand.

```
muxrun up
  → Spawn(configPath, groupName)
    → exec.Command(self, "_daemon", "--config", ..., "--group", ...)
    → Setsid: true      ← New session, detached from parent
    → stdin/stdout/stderr → /dev/null
    → WritePID()         ← /tmp/muxrun/daemon-{group}.pid
```

With `Setsid: true`, the daemon survives after the `muxrun up` command exits.

### Debouncer

File change events fire in rapid succession (editor temp files, renames, etc.). The debouncer uses a trailing-edge debounce pattern to coalesce them.

```
File events:  --A--B----C---------→
Timer (500ms): [==X [==X  [=========]→ callback fires
                ↑reset    ↑reset     ↑500ms elapsed
```

1. Each `Trigger()` cancels the existing timer and starts a new 500ms timer
2. If no new `Trigger()` occurs within 500ms, the callback fires
3. The callback runs `respawn-window -k` on the tmux window, which kills the running process and starts the command again — this also revives a pane that `remain-on-exit` left dead after a crash
4. `sync.Mutex` protects timer operations for thread safety

### Restart supervisor

tmux cannot push pane deaths to a client, so the daemon polls `list-windows` once a second per group and decides what to do with each `restart = "on-failure"` app from the pane's dead status and signal.

```
tick (1s)
  → pane alive         → nothing (10s alive resets the retry budget)
  → exit 0             → nothing
  → SIGINT/TERM/HUP/QUIT → nothing (the user stopped it)
  → exit != 0, SIGKILL, other signals
      → within backoff (1s, 2s, 4s, ... 30s)? → wait
      → 5 consecutive failures?               → give up, mark the window failed
      → otherwise                             → respawn-window -k
```

The decision is a pure function (`decide` in `supervisor.go`) so the policy is testable without a tmux server.

Per-app state lives in two places. Consecutive failures, the backoff deadline and the give-up flag are daemon memory. The restart count and the give-up mark are written to the window as `@muxrun_restarts` and `@muxrun_restart_state` user options, which `muxrun ps` reads in the same `list-windows` call it already makes: they live and die with the window, so `muxrun up` resets them without any file to clean up.

Watch restarts go through the same supervisor, which serializes them against failure restarts on a per-group mutex and gives a rebuilt app a fresh retry budget.

---

## 5. Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Command-line argument error |
| 130 | User cancellation (Ctrl+C) |

---

## 6. Naming Conventions

### tmux Resources

- Session name: `muxrun-{group_name}`
- Window name: `{app_name}`

---

## 7. External Command Dependencies

| Command | Required | Purpose |
|---------|----------|---------|
| `tmux` | Yes | Session and window management |
