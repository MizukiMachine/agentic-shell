# agentic-shell

`agentic-shell` は、自然言語の要望からエージェント仕様を収集し、Claude Code 互換のエージェント定義 Markdown を生成する Go 製 CLI です。

現在の実装で主に使うコマンドは次の 2 つです。

- `spec-gather`: Step-back 質問で要件を収集し、`AgentSpec` を YAML/JSON に出力
- `generate`: `AgentSpec` から `.claude/agents/*.md` を生成

加えて、仕様解析からスキル補完までを段階実行できる pipeline サブコマンドも使えます。

- `parse`: Markdown/YAML の仕様を共通 Envelope に変換
- `extract`: 仕様から要件と必要スキルを抽出
- `skill-scan`: 既存スキルディレクトリを走査
- `match`: 必要スキルと既存スキルを照合
- `skill-gen`: 不足スキルのプレースホルダー生成計画を作成
- `output`: `--skills-dir` 配下へ不足スキルの `SKILL.md` を書き出し

詳細な構造は [ARCHITECTURE.md](./ARCHITECTURE.md) を参照してください。

## インストール

### 1. ソースからビルド

```bash
git clone https://github.com/MizukiMachine/agentic-shell.git
cd agentic-shell
go build -o agentic-shell ./cmd/agentic-shell
```

### 2. `go install`

```bash
go install github.com/MizukiMachine/agentic-shell/cmd/agentic-shell@latest
```

### 3. Makefile を使う

```bash
make build
```

生成物は `bin/agentic-shell` に出力されます。

## 使用例

### 1. 仕様ファイルを作る

`spec.yaml` をカレントディレクトリに出したい場合は、`--output-dir .` を付けます。

```bash
./agentic-shell --output-dir . spec-gather --quick --output spec.yaml "code review agent"
```

### 2. 仕様ファイルからエージェントを生成する

```bash
./agentic-shell --output-dir . generate --from spec.yaml
```

生成先は `./.claude/agents/<agent-name>.md` です。

### 3. 1 回で仕様収集から生成まで進める

```bash
./agentic-shell generate "code review agent"
```

### 4. バージョン確認

```bash
./agentic-shell version
```

### 5. pipeline サブコマンドをつなげて不足スキルを生成する

```bash
./agentic-shell parse spec.md \
  | ./agentic-shell extract \
  | ./agentic-shell skill-scan --skills-dir .claude/skills \
  | ./agentic-shell match --skills-dir .claude/skills \
  | ./agentic-shell skill-gen --skills-dir .claude/skills \
  | ./agentic-shell output --skills-dir .claude/skills
```

`--skills-dir custom/skills` のように指定すると、`output` は `custom/skills/<skill-name>/SKILL.md` へ書き出します。

## 設定方法

設定は次の優先順で反映されます。

1. CLI フラグ
2. 環境変数
3. 設定ファイル
4. デフォルト値

### 設定ファイル

デフォルトでは `.agentic-shell.yaml` を探索します。明示的に指定する場合は `--config` を使います。

```bash
./agentic-shell --config ./agentic-shell.yaml --output-dir . spec-gather --output spec.yaml "documentation agent"
```

設定例:

```yaml
llm:
  claude_path: "claude"
  timeout: "2m"
  max_retries: 3

output:
  directory: ".claude/agents"
  format: "markdown"
  overwrite: false

gathering:
  confidence_threshold: 0.85
  max_question_rounds: 5

generation:
  default_model: "claude-sonnet-4-6"
  default_temperature: 0.7
```

### 環境変数

プレフィックスは `AGENTIC_` です。例:

```bash
export AGENTIC_OUTPUT_DIRECTORY=.
export AGENTIC_OUTPUT_OVERWRITE=true
export AGENTIC_GATHERING_CONFIDENCE_THRESHOLD=0.85
export AGENTIC_GENERATION_DEFAULT_MODEL=claude-sonnet-4-6
```

主なキー:

- `AGENTIC_LLM_CLAUDE_PATH`
- `AGENTIC_LLM_TIMEOUT`
- `AGENTIC_OUTPUT_DIRECTORY`
- `AGENTIC_OUTPUT_OVERWRITE`
- `AGENTIC_GATHERING_CONFIDENCE_THRESHOLD`
- `AGENTIC_GATHERING_MAX_QUESTION_ROUNDS`
- `AGENTIC_GENERATION_DEFAULT_MODEL`
- `AGENTIC_GENERATION_DEFAULT_TEMPERATURE`

## テスト

通常のテスト:

```bash
go test ./...
```

`cmd/agentic-shell` には実バイナリをビルドして実行する E2E テストが含まれています。現在のシナリオは次の 2 つです。

- `spec-gather --quick` で `spec.yaml` を出力できる
- `generate --from spec.yaml` で `.claude/agents/*.md` を生成できる

## 開発

主なコマンド:

- `make build`
- `make test`
- `make coverage`
- `make fmt`
- `make lint`

## プロジェクト構造

```text
agentic-shell/
├── cmd/agentic-shell/   # CLI エントリーポイントと E2E テスト
├── internal/cli/        # Cobra コマンド実装
├── internal/spec/       # 仕様収集と検証
├── internal/agent/      # エージェント定義生成
├── internal/config/     # 設定ロード
├── pkg/types/           # AgentSpec / IntentSpace / ClaudeAgentDefinition
└── docs/diagrams/       # 補助資料の図
```
