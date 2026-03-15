# agentic-shell

`agentic-shell` は、AIエージェント開発文脈のプロンプト、エージェント定義、スキル定義などを動的生成する Go 製 CLI です。
実行バイナリ名は `ags` です。

## インストール

```bash
go install github.com/MizukiMachine/agentic-shell/cmd/ags@latest
```

## 使い方

```bash
# 引数なしで対話モード
ags spec-gather
ags generate

# 引数を指定して直接実行
ags spec-gather "コードレビューエージェント" --output spec.yaml
ags generate --from spec.yaml

# 一発で生成
ags generate "ドキュメント生成エージェント"
```

## 主なコマンド

| コマンド | 説明 |
|---------|------|
| `spec-gather` | 要件を収集してAgentSpecを出力 |
| `generate` | AgentSpecからエージェント定義を生成 |
| `parse` | 仕様を構造化 |
| `extract` | 要件とスキルを抽出 |
| `skill-scan` | 既存スキルを走査 |

## 設定

`.ags.yaml` で設定（詳細は `ags --help` 参照）。

## 開発

```bash
make build   # ビルド
make test    # テスト
```
