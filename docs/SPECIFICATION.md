# muxrun 仕様書

## 概要

muxrun は tmux を利用して複数のアプリケーションをグループ単位で管理・起動する CLI ツール。

## 基本概念

### 構造

```
muxrun
├── Group A (= tmux session)
│   ├── App 1 (= tmux window)
│   └── App 2 (= tmux window)
└── Group B (= tmux session)
    └── App 3 (= tmux window)
```

- **グループ**: tmux セッションに対応。関連するアプリケーションをまとめる単位
- **アプリケーション**: tmux ウィンドウに対応。実際に実行されるプロセス

### ディレクトリ指定

アプリケーションごとに `dir` を指定する（必須）。CLI の `--dir` オプションで上書き可能。

### watch 機能

- 指定したアプリケーションの実行ディレクトリ以下を監視
- ファイル変更を検知するとアプリケーションを自動再起動
- アプリケーション単位で `watch = true` を指定して有効化

---

## 設定ファイル

### 場所

```
~/.config/muxrun/config.toml
```

### 構造

```toml
# グループ定義（1つ以上必須）
[[group]]
name = "backend"            # グループ名（必須、tmux セッション名になる）

  # アプリケーション定義（1つ以上必須）
  [[group.app]]
  name = "api"              # アプリ名（必須、tmux ウィンドウ名になる）
  cmd = "go run main.go"    # 実行コマンド（必須）
  dir = "~/projects/myapp/cmd/api"  # 実行ディレクトリ（必須）
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

### フィールド定義

#### グループ (`[[group]]`)

| フィールド | 型 | 必須 | 説明 |
|-----------|------|------|------|
| `name` | string | Yes | グループ名。tmux セッション名として使用 |

#### アプリケーション (`[[group.app]]`)

| フィールド | 型 | 必須 | 説明 |
|-----------|------|------|------|
| `name` | string | Yes | アプリ名。tmux ウィンドウ名として使用 |
| `cmd` | string | Yes | 実行コマンド |
| `dir` | string | Yes | 実行ディレクトリ |
| `watch` | bool \| object | No | ファイル監視設定。デフォルト false |

#### watch オプション

`watch` は以下の形式で指定可能:

- `watch = false` — 監視無効（デフォルト）
- `watch = { enabled = true }` — 監視有効（除外パターンなし）
- `watch = { enabled = true, exclude = [...] }` — 監視有効（除外パターンあり）

| フィールド | 型 | 必須 | 説明 |
|-----------|------|------|------|
| `enabled` | bool | Yes | ファイル監視の有効/無効 |
| `exclude` | string[] | No | 除外するファイルパターン（正規表現）。デフォルト空 |

### ディレクトリの解決順序

アプリケーションの実行ディレクトリ:

1. `--dir <path>` が指定されている → その値を使用
2. 未指定 → config のアプリの `dir` を使用

---

## CLI コマンド

### `muxrun check`

設定ファイルの構文と必須項目を検証する。

```bash
$ muxrun check
config: syntax is ok
config: test is successful

$ muxrun check
config: syntax error at line 15: unexpected character
config: test failed
```

**検証内容:**
- TOML 構文の正当性
- 必須フィールドの存在（グループ名、アプリ名、アプリ dir、cmd）
- グループが1つ以上存在すること
- 各グループにアプリが1つ以上存在すること
- グループ名・アプリ名の重複がないこと
- グループ名・アプリ名が命名規則に従っていること

### `muxrun ps`

起動中のアプリケーション一覧を表示。

```
$ muxrun ps
GROUP       APP       STATUS    PID
backend     api       running   12345
backend     worker    running   12346
frontend    dev       stopped   -
```

### `muxrun up`

アプリケーションを起動。

```bash
# 全グループ・全アプリを起動
muxrun up

# 特定グループの全アプリを起動
muxrun up backend

# 複数グループを起動
muxrun up backend frontend

# CLI でディレクトリを上書き指定
muxrun up backend --dir ~/projects/other-app

# fzf でインタラクティブにアプリを選択
muxrun up --interactive
muxrun up -i
```

**オプション:**

| オプション | 短縮形 | 説明 |
|-----------|--------|------|
| `--dir <path>` | なし | 実行ディレクトリを明示的に指定（config の dir を上書き） |
| `--interactive` | `-i` | fzf でアプリを対話的に選択 |

**位置引数:** `[group...]` — 対象グループ名を指定（省略時は全グループ）

**挙動:**
- 引数なし: 全グループ・全アプリを起動
- グループ名を指定: そのグループ内の全アプリを起動
- 複数グループ名を指定: 各グループ内の全アプリを起動
- `--dir` 指定: config の dir を上書き
- `--interactive` 指定: fzf で対象を選択（複数選択可）
- 既に起動中のアプリを指定: **エラー**

### `muxrun down`

アプリケーションを停止。

```bash
# 全グループ・全アプリを停止
muxrun down

# 特定グループの全アプリを停止
muxrun down backend

# 複数グループを停止
muxrun down backend frontend

# fzf でインタラクティブにアプリを選択
muxrun down --interactive
muxrun down -i
```

**オプション:**

| オプション | 短縮形 | 説明 |
|-----------|--------|------|
| `--interactive` | `-i` | fzf でアプリを対話的に選択 |

**位置引数:** `[group...]` — 対象グループ名を指定（省略時は全グループ）

**挙動:**
- 引数なし: 全グループ・全アプリを停止、全セッション終了
- グループ名を指定: そのグループ内の全アプリを停止、セッション終了
- 複数グループ名を指定: 各グループの全アプリを停止、セッション終了
- `--interactive` 指定: fzf で対象を選択（複数選択可）
- 停止済みのアプリを指定: 無視（正常終了）

---

## tmux セッション管理

### 命名規則

- セッション名: `muxrun-{group_name}`
- ウィンドウ名: `{app_name}`

### ライフサイクル

1. `muxrun up group`: セッション `muxrun-group` を作成（存在しなければ）
2. 各アプリに対応するウィンドウを作成し、コマンドを実行
3. `muxrun down group`: 全ウィンドウを閉じ、セッションを終了
4. `muxrun down group app`: 該当ウィンドウのみ閉じる。最後のウィンドウならセッションも終了

---

## watch 機能詳細

### 監視対象

- アプリケーションの実行ディレクトリ以下の全ファイル
- 隠しファイル・ディレクトリは除外（`.git`, `node_modules` など）
- `exclude` で指定した正規表現パターンにマッチするファイルは除外

### exclude パターン

- 正規表現（Go の `regexp` パッケージ構文）で指定
- 複数パターンを配列で指定可能
- ファイルの相対パス（実行ディレクトリからの相対）に対してマッチング
- いずれかのパターンにマッチしたファイルは監視対象から除外

```toml
watch = { enabled = true, exclude = [
  "_test\\.go$",      # テストファイルを除外
  "mock_.*\\.go$",    # モックファイルを除外
  "testdata/",        # testdata ディレクトリ以下を除外
  "\\.tmp$",          # .tmp ファイルを除外
] }
```

### 再起動の流れ

1. ファイル変更を検知
2. デバウンス処理（連続した変更を1回にまとめる）
3. 現在のプロセスに SIGTERM を送信
4. 一定時間後に応答がなければ SIGKILL
5. コマンドを再実行

---

## エラーハンドリング

| 状況 | 挙動 |
|------|------|
| config ファイルが存在しない | エラー終了 |
| config の構文エラー | エラー終了、行番号を表示 |
| 指定されたグループが存在しない | エラー終了 |
| 指定されたアプリが存在しない | エラー終了 |
| 存在しないグループを指定 | エラー終了 |
| 起動中のアプリに `up` を実行 | エラー終了 |
| 停止中のアプリに `down` を実行 | 正常終了（何もしない） |
| fzf がキャンセルされた | エラー終了 |
| fzf が利用不可 | エラー終了 |
| tmux が利用不可 | エラー終了 |

---

## 制約事項

- グループは1つ以上必須
- 各グループにはアプリケーションが1つ以上必須
- グループ名・アプリ名は英数字とハイフン、アンダースコアのみ使用可能
- 同一グループ内でアプリ名の重複は不可
- グループ名の重複は不可
