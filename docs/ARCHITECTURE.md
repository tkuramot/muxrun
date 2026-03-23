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
│   ├── daemon/     # File-watch daemon spawning and PID management
│   ├── runner/     # App start/stop/status orchestration
│   ├── selector/   # fzf-based interactive selection
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
│ (Application)            │  │ (File Watch Daemon)         │
│ - App start/stop         │  │ - Daemon process spawning   │
│ - Concurrent app control │  │ - PID file management       │
└───────┬──────────────────┘  │ - File change → app restart │
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

- `cmd/` → `internal/runner/`, `internal/daemon/`, `internal/config/`, `internal/selector/`
- `internal/runner/` → `internal/tmux/`, `internal/config/`
- `internal/daemon/` → `internal/tmux/`, `internal/watcher/`, `internal/config/`, `internal/runner/`
- Each `internal/` package is loosely coupled (`daemon` only references `runner`'s `process.go` utilities)

---

## 4. Daemon Architecture

### Independent daemon process per group

`muxrun up` spawns a separate daemon process for each group that requires file watching.

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
3. The callback sends `C-c` to the tmux window → waits 100ms → resends the command to restart the process
4. `sync.Mutex` protects timer operations for thread safety

---

## 5. Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Command-line argument error |
| 130 | User cancellation (Ctrl+C, fzf cancel) |

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
| `fzf` | No | Only for the `--interactive` option |
