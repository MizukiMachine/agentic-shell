# agentic-shell

AIエージェント統合シェル - 複数のAIエージェントを統合管理するターミナルベースのシェルアプリケーション

## 概要

`agentic-shell` は、Claude、GPT、Gemini などの AI エージェントと対話しながら開発作業を効率化するための CLI ツールです。

## 機能

- 複数AIエージェントの統合管理
- ターミナルベースのユーザーインターフェース
- 設定ファイルによるカスタマイズ
- 拡張可能なエージェントアーキテクチャ

## インストール

### ソースからのビルド

```bash
# リポジトリをクローン
git clone https://github.com/MizukiMachine/agentic-shell.git
cd agentic-shell

# 依存パッケージをインストール
make deps

# ビルド
make build
```

### Go install を使用

```bash
go install github.com/MizukiMachine/agentic-shell/cmd/agentic-shell@latest
```

## 使用方法

### 基本的な使い方

```bash
# ヘルプを表示
agentic-shell --help

# バージョンを表示
agentic-shell version
```

## 開発

### 必要なツール

- Go 1.21 以上
- Make

### Make コマンド

| コマンド | 説明 |
|---------|------|
| `make build` | 本番用ビルド |
| `make dev` | 開発用ビルド |
| `make test` | テスト実行 |
| `make coverage` | テストカバレッジ |
| `make lint` | 静的解析 |
| `make fmt` | フォーマット |
| `make clean` | クリーンアップ |

## プロジェクト構造

```
agentic-shell/
├── cmd/
│   └── agentic-shell/    # CLI エントリーポイント
│       └── main.go
├── internal/
│   ├── config/           # 設定管理
│   ├── agent/            # エージェント実装
│   └── tui/              # ターミナルUI
├── pkg/                  # 公開パッケージ
├── Makefile
├── go.mod
└── README.md
```

## ライセンス

MIT License

## 貢献

プルリクエストや Issue は歓迎します。

## 作者

MizukiMachine
