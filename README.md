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
