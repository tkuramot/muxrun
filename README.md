# muxrun

tmux を利用して複数のアプリケーションをグループ単位で管理・起動する CLI ツール。

## 概要

muxrun はグループとアプリケーションの 2 階層で複数プロセスを管理します。グループは tmux セッション、アプリケーションは tmux ウィンドウに対応し、設定ファイルに定義したコマンドを一括で起動・停止できます。

```
muxrun
├── Group A (= tmux session)
│   ├── App 1 (= tmux window)
│   └── App 2 (= tmux window)
└── Group B (= tmux session)
    └── App 3 (= tmux window)
```

## 必要要件

- Go 1.22+
- tmux 3.0+
- fzf（`--interactive` オプション使用時のみ）

## インストール

```bash
go install github.com/tkuramot/muxrun@latest
```

## 設定

設定ファイルを `~/.config/muxrun/config.toml` に作成します。

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

### フィールド

| フィールド | 型 | 必須 | 説明 |
|-----------|------|------|------|
| `group.name` | string | Yes | グループ名（tmux セッション名） |
| `group.app.name` | string | Yes | アプリ名（tmux ウィンドウ名） |
| `group.app.cmd` | string | Yes | 実行コマンド |
| `group.app.dir` | string | Yes | 実行ディレクトリ |
| `group.app.watch` | object | No | ファイル監視設定（デフォルト: 無効） |

### watch 設定

```toml
# 監視有効
watch = { enabled = true }

# 監視有効 + 除外パターン（正規表現）
watch = { enabled = true, exclude = ["_test\\.go$", "testdata/"] }
```

ファイル変更を検知するとアプリケーションを自動再起動します。

## 使い方

### 設定ファイルの検証

```bash
muxrun check
```

### アプリケーションの起動

```bash
# 全グループ・全アプリを起動
muxrun up

# 特定グループの全アプリを起動
muxrun up -g backend

# 特定グループの特定アプリのみ起動
muxrun up -g backend -a api

# ディレクトリを上書き指定
muxrun up -g backend --dir ~/projects/other-app

# fzf でインタラクティブに選択
muxrun up -i
```

### アプリケーションの停止

```bash
# 全グループ・全アプリを停止
muxrun down

# 特定グループの全アプリを停止
muxrun down -g backend

# 特定アプリのみ停止
muxrun down -g backend -a api

# fzf でインタラクティブに選択
muxrun down -i
```

### ステータス確認

```bash
$ muxrun ps
GROUP       APP       STATUS    PID
backend     api       running   12345
backend     worker    running   12346
frontend    dev       stopped   -
```

## tmux セッションの操作

muxrun が起動したプロセスは tmux セッション内で動作しています。セッション名は `muxrun-<グループ名>` の形式です。

### セッションへのアタッチ

```bash
# backend グループのセッションにアタッチ
tmux attach -t muxrun-backend
```

アタッチ後は通常の tmux 操作が可能です。

### tmux 基本操作（プレフィックスキーはデフォルトで `Ctrl-b`）

| 操作 | キー | 説明 |
|------|------|------|
| ウィンドウ切り替え（次） | `Ctrl-b` `n` | 次のアプリ（ウィンドウ）に移動 |
| ウィンドウ切り替え（前） | `Ctrl-b` `p` | 前のアプリ（ウィンドウ）に移動 |
| ウィンドウ一覧 | `Ctrl-b` `w` | ウィンドウを一覧表示して選択 |
| デタッチ | `Ctrl-b` `d` | セッションから抜ける（プロセスは継続） |
| スクロールモード | `Ctrl-b` `[` | ログをスクロールして確認（`q` で終了） |

### セッション・ウィンドウの確認

```bash
# muxrun が管理しているセッション一覧
tmux list-sessions | grep muxrun-

# 特定セッションのウィンドウ（アプリ）一覧
tmux list-windows -t muxrun-backend
```

### 注意事項

- セッションやウィンドウの停止には `muxrun down` を使用してください。`tmux kill-session` で直接終了すると muxrun の状態管理と不整合が生じる場合があります。
- アタッチ中にウィンドウ内でプロセスを手動停止した場合も、`muxrun ps` のステータスに反映されます。

## 開発

```bash
# ユニットテスト
go test ./...

# 統合テスト
go test -tags=integration ./...

# E2E テスト
go test -tags=e2e ./...
```

## ライセンス

MIT
