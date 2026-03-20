# muxrun Architecture

## 1. Tech Stack

### Language & Runtime

- **Go 1.26+**

### External Dependencies

| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| `pelletier/go-toml/v2` | v2.2.0 | TOML parsing | TOML 1.0 compliant, detailed error messages, actively maintained |
| `urfave/cli/v2` | v2.27.0 | CLI framework | Zero dependencies, built-in subcommands, aliases, and help generation |
| `fsnotify/fsnotify` | v1.7.0 | File watching | De facto standard, cross-platform support |

### Standard Library Usage

| Feature | Package |
|---------|---------|
| Regex (exclude patterns) | `regexp` |
| Path expansion (`~` → home directory) | `os.UserHomeDir()` |
| Table output (ps command) | `text/tabwriter` |
| External command execution (tmux, fzf) | `os/exec` |
| Testing | `testing`, `cmp` |

---

## 2. Directory Structure

```
muxrun/
├── main.go                     # Entry point (minimal)
├── go.mod
├── go.sum
│
├── cmd/                        # CLI command definitions
│   ├── root.go                 # Root command, shared config
│   ├── check.go                # check subcommand
│   ├── ps.go                   # ps subcommand
│   ├── up.go                   # up subcommand
│   ├── down.go                 # down subcommand
│   └── daemon.go               # _daemon subcommand (hidden)
│
├── internal/                   # Private packages
│   ├── config/                 # Configuration
│   │   ├── config.go           # Config struct, TOML parsing
│   │   ├── config_test.go
│   │   ├── validator.go        # Validation logic
│   │   └── validator_test.go
│   │
│   ├── tmux/                   # tmux operations
│   │   ├── client.go           # tmux command wrapper
│   │   ├── client_test.go
│   │   ├── session.go          # Session management
│   │   └── window.go           # Window management
│   │
│   ├── watcher/                # File watching
│   │   ├── watcher.go          # File watch implementation
│   │   ├── watcher_test.go
│   │   ├── debouncer.go        # Debounce logic
│   │   └── filter.go           # Exclude pattern filter
│   │
│   ├── daemon/                 # File watch daemon
│   │   ├── daemon.go           # Daemon spawning & main loop
│   │   └── pidfile.go          # PID file management
│   │
│   ├── runner/                 # Application execution management
│   │   ├── runner.go           # Up/Down/Status orchestration
│   │   ├── runner_test.go
│   │   └── process.go          # Process management (SIGTERM/SIGKILL)
│   │
│   ├── selector/               # fzf integration
│   │   ├── fzf.go              # fzf interactive selection
│   │   └── fzf_test.go
│   │
│   └── ui/                     # Output formatting
│       └── table.go            # Table format output
│
├── docs/                       # Documentation
│   ├── SPECIFICATION.md
│   └── ARCHITECTURE.md
│
└── testdata/                   # Test fixtures
    ├── valid_config.toml
    ├── invalid_syntax.toml
    └── missing_required.toml
```

### Rationale

- **`cmd/`**: One file per subcommand. Clear separation of concerns improves maintainability.
- **`internal/`**: Go's language-level import restriction prevents external use. Allows free refactoring without API stability concerns.
- **Feature-based packages**: `config`, `tmux`, `watcher`, `daemon`, `runner`, `selector` — each package has a single responsibility.

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

## 4. Key Interfaces

### 4.1 Config

```go
// internal/config/config.go

type Config struct {
    Groups []Group
}

type Group struct {
    Name string
    Apps []App
}

type App struct {
    Name  string
    Cmd   string
    Dir   string
    Watch WatchConfig
}

type WatchConfig struct {
    Enabled bool
    Exclude []string
}

// Loader loads a config file
type Loader interface {
    Load(path string) (*Config, error)
}

// Validator validates a config
type Validator interface {
    Validate(cfg *Config) error
}
```

### 4.2 Tmux Client

```go
// internal/tmux/client.go

type Client interface {
    // Session operations
    HasSession(name string) (bool, error)
    NewSession(name string) error
    KillSession(name string) error
    ListSessions() ([]Session, error)

    // Window operations
    NewWindow(session, window, dir string) error
    KillWindow(session, window string) error
    ListWindows(session string) ([]Window, error)
    SendKeys(session, window, keys string) error

    // State queries
    GetPanePID(session, window string) (int, error)
}

type Session struct {
    Name    string
    Windows []Window
}

type Window struct {
    Name   string
    PID    int
    Active bool
}
```

### 4.3 Watcher

```go
// internal/watcher/watcher.go

type Watcher interface {
    Watch(dir string, excludePatterns []string) (<-chan Event, error)
    Stop() error
}

type Event struct {
    Path      string
    Operation Op
    Time      time.Time
}

type Op int

const (
    Create Op = iota
    Write
    Remove
    Rename
)
```

### 4.4 Runner

```go
// internal/runner/runner.go

type Runner interface {
    Up(ctx context.Context, opts UpOptions) error
    Down(ctx context.Context, opts DownOptions) error
    Status() ([]AppStatus, error)
}

type UpOptions struct {
    GroupName   string
    AppName     string
    DirOverride string
}

type DownOptions struct {
    GroupName string
    AppName   string
}

type AppStatus struct {
    Group  string
    App    string
    Status Status
    PID    int
}

type Status string

const (
    StatusRunning Status = "running"
    StatusStopped Status = "stopped"
)
```

### 4.5 Selector

```go
// internal/selector/fzf.go

type Selector interface {
    SelectGroups(groups []string) ([]string, error)
    SelectApps(apps []AppOption) ([]AppOption, error)
}

type AppOption struct {
    Group string
    App   string
}
```

---

## 5. Daemon Architecture

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

## 6. Error Handling

### Sentinel Errors

```go
var (
    ErrConfigNotFound     = errors.New("config file not found")
    ErrConfigSyntax       = errors.New("config syntax error")
    ErrConfigValidation   = errors.New("config validation error")
    ErrGroupNotFound      = errors.New("group not found")
    ErrAppNotFound        = errors.New("app not found")
    ErrAppAlreadyRunning  = errors.New("app already running")
    ErrTmuxNotAvailable   = errors.New("tmux is not available")
    ErrFzfNotAvailable    = errors.New("fzf is not available")
    ErrFzfCancelled       = errors.New("fzf selection cancelled")
)
```

### Custom Error Types

```go
type ConfigSyntaxError struct {
    Line    int
    Column  int
    Message string
}

func (e *ConfigSyntaxError) Error() string {
    return fmt.Sprintf("syntax error at line %d, column %d: %s", e.Line, e.Column, e.Message)
}

func (e *ConfigSyntaxError) Unwrap() error {
    return ErrConfigSyntax
}
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Command-line argument error |
| 130 | User cancellation (Ctrl+C, fzf cancel) |

---

## 7. Testing Strategy

### Test Levels

| Level | Target | Build Tag |
|-------|--------|-----------|
| Unit tests | Individual package functions | None |
| Integration tests | Cross-package interactions | `integration` |
| E2E tests | Real tmux usage | `e2e` |

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# E2E tests
go test -tags=e2e ./...
```

### Mock Strategy

- Mock implementations for the `internal/tmux/Client` interface
- Inject mocks via DI during testing

---

## 8. Naming Conventions

### tmux Resources

- Session name: `muxrun-{group_name}`
- Window name: `{app_name}`

### Config File

- Location: `~/.config/muxrun/config.toml`
- Group and app names: alphanumeric characters, hyphens, and underscores only

---

## 9. External Command Dependencies

| Command | Required | Purpose |
|---------|----------|---------|
| `tmux` | Yes | Session and window management |
| `fzf` | No | Only for the `--interactive` option |

### Version Requirements

- tmux: 3.0 or later (recommended)
