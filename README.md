# agentic-shell

`agentic-shell` は、AIエージェント開発文脈のプロンプト、エージェント定義、スキル定義などを動的生成する Go 製 CLI です。
実行バイナリ名は `ags` です。

## 前提条件

- Go 1.21+
- GLM API Key（動的質問生成機能を使用する場合）

## インストール

```bash
go install github.com/MizukiMachine/agentic-shell/cmd/ags@latest
```

## 環境変数の設定

動的質問生成機能（LLM）を使用する場合、GLM API キーを設定してください：

```bash
export GLM_API_KEY=your_api_key_here
```

API キーは [GLM AI Platform](https://open.bigmodel.cn/) から取得できます。

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

# LLMを使わず固定質問モードで収集
ags spec-gather --no-llm "テストエージェント"

# モデルを上書き
ags spec-gather --llm-model glm-4 "高速エージェント"
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

### 設定例

```yaml
llm:
  provider: glm
  base_url: https://open.bigmodel.cn/api/paas/v4/
  model: glm-4-flash
  timeout: 2m
  max_retries: 3

gathering:
  confidence_threshold: 0.85
  max_question_rounds: 5
  use_llm_questions: true

output:
  directory: .claude/agents
  format: markdown
```

### spec-gather コマンドのフラグ

| フラグ | 説明 |
|--------|------|
| `--no-llm` | 固定質問モードを使用（LLMを使わない） |
| `--llm-model` | LLMモデル名を上書き |
| `--output, -o` | 出力ファイルパス |
| `--format, -f` | 出力形式（yaml/json） |
| `--quick, -q` | クイックモード（低信頼度でも継続） |
| `--timeout, -t` | 全体タイムアウト（秒） |

## 開発

```bash
make build   # ビルド
make test    # テスト
```
