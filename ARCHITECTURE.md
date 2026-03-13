# ARCHITECTURE

## 目的

`agentic-shell` は、曖昧な自然言語要求を段階的に構造化し、最終的に Claude Code 互換のエージェント定義 Markdown へ変換する CLI です。

現時点で実装されている主経路は次の 2 本です。

1. `spec-gather`: 入力から `AgentSpec` を生成する
2. `generate`: `AgentSpec` からエージェントファイルを生成する

## 実行フロー

### `spec-gather`

```text
ユーザー入力
  -> internal/cli/spec_gather.go
  -> internal/spec.Gatherer
  -> Step-back 質問で回答収集
  -> internal/spec.ValidateWithThreshold
  -> YAML / JSON 出力
```

### `generate`

```text
ユーザー入力 or --from spec.yaml
  -> internal/cli/generate.go
  -> internal/spec.Gatherer または spec ファイル読込
  -> internal/agent.Generator
  -> ClaudeAgentDefinition
  -> Markdown テンプレート描画
  -> .claude/agents/<name>.md
```

## ディレクトリ構造

```text
cmd/
  agentic-shell/
    main.go              # エントリーポイント
    main_test.go         # CLI バイナリ前提の基本テスト
    e2e_test.go          # spec-gather / generate の E2E テスト

internal/
  cli/
    root.go              # ルートコマンドとグローバルフラグ
    spec_gather.go       # spec-gather コマンド
    generate.go          # generate コマンド
    version.go           # version コマンド
  spec/
    gatherer.go          # 対話収集ロジック
    confidence.go        # 信頼度計算
    validator.go         # 必須項目と閾値検証
  agent/
    generator.go         # AgentSpec -> ClaudeAgentDefinition 変換
    templates.go         # Markdown テンプレート
  config/
    config.go            # デフォルト値と設定構造体
    loader.go            # Viper ベースの設定ロード

pkg/
  types/
    intent.go            # IntentSpace の 4 次元型
    agent_spec.go        # 中間表現 AgentSpec
    agent_def.go         # 最終表現 ClaudeAgentDefinition
```

## レイヤ責務

### 1. CLI レイヤ

`internal/cli` は Cobra ベースの入出力境界です。

- フラグ解析
- 設定読込
- `stdin/stdout/stderr` の配線
- ユースケース呼び出し

ビジネスロジックは `internal/spec` と `internal/agent` に寄せています。

### 2. 仕様収集レイヤ

`internal/spec` は、曖昧な要求を `AgentSpec` に落とし込む責務を持ちます。

- `Gatherer`: 対話型の質問ループ
- `ConfidenceCalculator`: 情報の埋まり具合から信頼度を算出
- `ValidateWithThreshold`: 必須フィールドと信頼度閾値を検証

`spec-gather --quick` は、途中エラーがあっても部分的に収集できた `AgentSpec` を優先するための運用モードです。

### 3. 生成レイヤ

`internal/agent` は `AgentSpec` を Claude Code 用の Markdown に変換します。

- `Generate`: `AgentSpec` から `ClaudeAgentDefinition` を生成
- `RenderMarkdown`: frontmatter と system prompt を Markdown 化
- `MarkdownFileName`: エージェント名を安全なファイル名へ正規化

出力先は `buildAgentOutputPath` により決まり、通常は `output.directory/.claude/agents/<name>.md` です。`output.directory` 自体が `.claude/agents` で終わる場合はその配下へそのまま出力します。

### 4. 設定レイヤ

`internal/config` は次の順で値を合成します。

1. デフォルト値
2. 設定ファイル
3. 環境変数
4. CLI フラグ上書き

主要設定:

- `llm`: CLI パス、タイムアウト、リトライ
- `output`: 出力ディレクトリ、フォーマット、上書き可否
- `gathering`: 信頼度閾値、最大質問回数
- `generation`: デフォルトモデル、温度

## 型定義

### `IntentSpace`

`pkg/types/intent.go` は、ユーザー意図を 4 次元で持つコアモデルです。

- `Goals`: 何を達成したいか
- `Preferences`: 品質/速度、コスト/性能、自動化/統制、リスク許容度
- `Objectives`: 機能要件、非機能要件、制約、品質要件
- `Modality`: 期待する出力形式。テキスト、データ、コードなど

`Gatherer` は回答内容からこの空間を徐々に埋め、信頼度計算の材料にします。

### `AgentSpec`

`pkg/types/agent_spec.go` の `AgentSpec` は中間表現です。

- `Metadata`: 名前、説明、タグ、版
- `Intent`: 上記 `IntentSpace`
- `Capabilities`: エージェントの能力
- `Skills`: エージェントが持つ技能
- `Tools`: 利用可能ツールの抽象定義
- `BehaviorRules`, `KnowledgeSources`
- `Communication`, `Performance`, `Security`

`spec-gather` の出力ファイルはこの型に対応します。YAML/JSON のどちらでも扱えます。

### `ClaudeAgentDefinition`

`pkg/types/agent_def.go` の `ClaudeAgentDefinition` は最終出力です。

- `Metadata`
- `Prompt`
- `Model`
- `Context`
- `Tools`
- `Safety`
- `Output`
- `Logging`
- `Metrics`

`generate` は `AgentSpec` をこの型へ変換した後、Markdown frontmatter と本文にレンダリングします。

## テスト戦略

### ユニットテスト

- `internal/spec/*_test.go`: 収集ロジック、信頼度、検証
- `internal/agent/*_test.go`: 生成ロジック、テンプレート出力
- `internal/config/*_test.go`: 設定ロード
- `pkg/types/*_test.go`: 型の妥当性

### E2E テスト

`cmd/agentic-shell/e2e_test.go` は、テスト中に `go build -o <temp>/agentic-shell ./cmd/agentic-shell` を実行し、その実バイナリを使って検証します。

現在のシナリオ:

1. `spec-gather --quick --output spec.yaml` が `AgentSpec` を出力する
2. `generate --from spec.yaml` が `.claude/agents/code-review-agent.md` を生成する

## 補足

`docs/diagrams/` には補助的な図がありますが、実装把握の一次情報は `internal/` と `pkg/types/` のコードです。実装変更時はこの文書と README を先に更新するとズレが小さくなります。
