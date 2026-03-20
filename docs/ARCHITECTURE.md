# muxrun アーキテクチャ設計書

## 1. 技術スタック

### 言語・ランタイム

- **Go 1.22+**

### 外部依存ライブラリ

| ライブラリ | バージョン | 用途 | 選定理由 |
|-----------|-----------|------|----------|
| `pelletier/go-toml/v2` | v2.2.0 | TOML パース | TOML 1.0 準拠、詳細なエラーメッセージ、活発なメンテナンス |
| `urfave/cli/v2` | v2.27.0 | CLI フレームワーク | ゼロ依存、サブコマンド・エイリアス・ヘルプ生成が組み込み済み |
| `fsnotify/fsnotify` | v1.7.0 | ファイル監視 | デファクトスタンダード、クロスプラットフォーム対応 |

### 標準ライブラリで対応する機能

| 機能 | 使用パッケージ |
|------|---------------|
| 正規表現（exclude パターン） | `regexp` |
| パス展開（`~` → ホームディレクトリ） | `os.UserHomeDir()` |
| テーブル出力（ps コマンド） | `text/tabwriter` |
| 外部コマンド実行（tmux, fzf） | `os/exec` |
| テスト | `testing`, `cmp` |

---

## 2. ディレクトリ構造

```
muxrun/
├── main.go                     # エントリーポイント（最小限）
├── go.mod
├── go.sum
│
├── cmd/                        # CLI コマンド定義
│   ├── root.go                 # ルートコマンド、共通設定
│   ├── check.go                # check サブコマンド
│   ├── ps.go                   # ps サブコマンド
│   ├── up.go                   # up サブコマンド
│   ├── down.go                 # down サブコマンド
│   └── daemon.go               # _daemon サブコマンド（hidden）
│
├── internal/                   # 非公開パッケージ
│   ├── config/                 # 設定ファイル関連
│   │   ├── config.go           # Config 構造体、TOML パース
│   │   ├── config_test.go
│   │   ├── validator.go        # バリデーションロジック
│   │   └── validator_test.go
│   │
│   ├── tmux/                   # tmux 操作
│   │   ├── client.go           # tmux コマンドラッパー
│   │   ├── client_test.go
│   │   ├── session.go          # セッション管理
│   │   └── window.go           # ウィンドウ管理
│   │
│   ├── watcher/                # ファイル監視
│   │   ├── watcher.go          # ファイル監視実装
│   │   ├── watcher_test.go
│   │   ├── debouncer.go        # デバウンス処理
│   │   └── filter.go           # exclude パターンフィルタ
│   │
│   ├── daemon/                 # ファイル監視 daemon
│   │   ├── daemon.go           # daemon のスポーン・メインループ
│   │   └── pidfile.go          # PID ファイル管理
│   │
│   ├── runner/                 # アプリケーション実行管理
│   │   ├── runner.go           # Up/Down/Status オーケストレーション
│   │   ├── runner_test.go
│   │   └── process.go          # プロセス管理（SIGTERM/SIGKILL）
│   │
│   ├── selector/               # fzf 連携
│   │   ├── fzf.go              # fzf インタラクティブ選択
│   │   └── fzf_test.go
│   │
│   └── ui/                     # 出力フォーマット
│       └── table.go            # テーブル形式出力
│
├── docs/                       # ドキュメント
│   ├── SPECIFICATION.md
│   └── ARCHITECTURE.md
│
└── testdata/                   # テスト用フィクスチャ
    ├── valid_config.toml
    ├── invalid_syntax.toml
    └── missing_required.toml
```

### 構造の根拠

- **`cmd/`**: サブコマンドごとにファイル分離。責務が明確で保守性向上
- **`internal/`**: Go の言語仕様により外部からインポート不可。API 安定性を気にせずリファクタリング可能
- **機能単位のパッケージ**: `config`, `tmux`, `watcher`, `daemon`, `runner`, `selector` と責務ごとに分離

---

## 3. レイヤーアーキテクチャ

```
┌─────────────────────────────────────────────────┐
│                   cmd/ (CLI Layer)              │
│  - コマンドライン引数のパース                      │
│  - ユーザー入力のバリデーション                    │
│  - 出力フォーマット                               │
│  - daemon のスポーン・停止制御                     │
└─────────────────────┬───────────────────────────┘
                      │
                      ▼
┌──────────────────────────┐  ┌────────────────────────────┐
│ internal/runner/         │  │ internal/daemon/            │
│ (Application)            │  │ (File Watch Daemon)         │
│ - アプリ起動・停止        │  │ - daemon プロセスのスポーン   │
│ - 複数アプリの並行制御    │  │ - PID ファイル管理           │
└───────┬──────────────────┘  │ - ファイル変更→アプリ再起動  │
        │                     └──────┬─────────────────────┘
        │                            │
        ▼                            ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│ internal/tmux │  │internal/watcher│ │internal/config│
│ - Session管理  │  │ - ファイル監視  │ │ - TOML パース  │
│ - Window管理   │  │ - デバウンス    │ │ - バリデーション│
└───────────────┘  └───────────────┘  └───────────────┘
```

### 依存方向

- `cmd/` → `internal/runner/`, `internal/daemon/`, `internal/config/`, `internal/selector/`
- `internal/runner/` → `internal/tmux/`, `internal/config/`
- `internal/daemon/` → `internal/tmux/`, `internal/watcher/`, `internal/config/`, `internal/runner/`
- 各 `internal/` パッケージは互いに疎結合（`daemon` は `runner` の `process.go` ユーティリティのみ参照）

---

## 4. 主要インターフェース

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

// Loader は設定ファイルを読み込む
type Loader interface {
    Load(path string) (*Config, error)
}

// Validator は設定を検証する
type Validator interface {
    Validate(cfg *Config) error
}
```

### 4.2 Tmux Client

```go
// internal/tmux/client.go

type Client interface {
    // セッション操作
    HasSession(name string) (bool, error)
    NewSession(name string) error
    KillSession(name string) error
    ListSessions() ([]Session, error)

    // ウィンドウ操作
    NewWindow(session, window, dir string) error
    KillWindow(session, window string) error
    ListWindows(session string) ([]Window, error)
    SendKeys(session, window, keys string) error

    // 状態取得
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

## 5. Daemon アーキテクチャ

### グループごとに独立した daemon プロセス

`muxrun up` でファイル監視が必要なグループごとに、個別の daemon プロセスをスポーンする設計を採用している。

```
muxrun up
  ├── daemon (group: frontend)   ← PID 1234
  └── daemon (group: backend)    ← PID 5678
```

**設計判断の理由:**

| 観点 | グループ単位 daemon（採用） | 単一 daemon |
|------|---------------------------|-------------|
| ライフサイクル管理 | `up`/`down` がグループ単位で完結。kill → respawn するだけ | 設定のホットリロードや部分更新ロジックが必要 |
| 障害分離 | 1 プロセスのクラッシュが他グループに波及しない | 全グループが道連れ |
| 実装の単純さ | `Spawn()` / `StopDaemon()` が PID ファイル1つで管理可能 | プロセス内でグループの動的追加・削除を管理する必要がある |

### スポーンの仕組み

`Spawn()` は自分自身の実行ファイルを hidden サブコマンド `_daemon` で再起動する。

```
muxrun up
  → Spawn(configPath, groupName)
    → exec.Command(self, "_daemon", "--config", ..., "--group", ...)
    → Setsid: true      ← 新セッションで親から切り離し
    → stdin/stdout/stderr → /dev/null
    → WritePID()         ← /tmp/muxrun/daemon-{group}.pid
```

`Setsid: true` により、`muxrun up` コマンドが終了しても daemon は生存し続ける。

### Debouncer（デバウンス処理）

ファイル変更イベントは短時間に大量発生する（エディタの一時ファイル作成・リネーム等）。Debouncer は trailing edge debounce パターンでこれを間引く。

```
ファイルイベント:  --A--B----C---------→
Timer (500ms):   [==X [==X  [=========]→ callback 発火
                  ↑リセット  ↑リセット   ↑500ms 経過
```

1. `Trigger()` のたびに既存 timer をキャンセルし、500ms の新 timer を開始
2. 500ms 間新たな `Trigger()` がなければ callback が発火
3. callback は tmux ウィンドウに `C-c` → 100ms 待機 → コマンド再送信でプロセスを再起動
4. `sync.Mutex` で timer 操作をスレッドセーフに保護

---

## 6. エラーハンドリング

### センチネルエラー

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

### カスタムエラー型

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

### 終了コード

| コード | 意味 |
|--------|------|
| 0 | 正常終了 |
| 1 | 一般的なエラー |
| 2 | コマンドライン引数エラー |
| 130 | ユーザーキャンセル（Ctrl+C, fzf キャンセル） |

---

## 7. テスト戦略

### テストレベル

| レベル | 対象 | ビルドタグ |
|--------|------|-----------|
| ユニットテスト | 各パッケージの関数 | なし |
| 統合テスト | パッケージ間連携 | `integration` |
| E2E テスト | 実際の tmux を使用 | `e2e` |

### 実行方法

```bash
# ユニットテスト
go test ./...

# 統合テスト
go test -tags=integration ./...

# E2E テスト
go test -tags=e2e ./...
```

### モック戦略

- `internal/tmux/Client` インターフェースに対してモック実装を用意
- テスト時に DI でモックを注入

---

## 8. 命名規則

### tmux リソース

- セッション名: `muxrun-{group_name}`
- ウィンドウ名: `{app_name}`

### 設定ファイル

- 場所: `~/.config/muxrun/config.toml`
- グループ名・アプリ名: 英数字、ハイフン、アンダースコアのみ

---

## 9. 外部コマンド依存

| コマンド | 必須 | 用途 |
|----------|------|------|
| `tmux` | Yes | セッション・ウィンドウ管理 |
| `fzf` | No | `--interactive` オプション使用時のみ |

### バージョン要件

- tmux: 3.0 以上（推奨）
