# agentic-shell Makefile
# ビルド、テスト、インストールなどのタスクを定義

# 変数定義
APP_NAME := agentic-shell
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"

# デフォルトターゲット
.PHONY: all
all: build

# ビルド
.PHONY: build
build:
	@echo "ビルド中... $(APP_NAME)"
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/agentic-shell
	@echo "ビルド完了: bin/$(APP_NAME)"

# 開発用ビルド（高速）
.PHONY: dev
dev:
	@echo "開発ビルド中..."
	$(GO) build $(GOFLAGS) -o bin/$(APP_NAME) ./cmd/agentic-shell
	@echo "開発ビルド完了"

# テスト実行
.PHONY: test
test:
	@echo "テスト実行中..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "テスト完了"

# テストカバレッジ表示
.PHONY: coverage
coverage: test
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "カバレッジレポート生成: coverage.html"

# インストール（GOPATH/binに配置）
.PHONY: install
install:
	@echo "インストール中..."
	$(GO) install $(LDFLAGS) ./cmd/agentic-shell
	@echo "インストール完了"

# クリーンアップ
.PHONY: clean
clean:
	@echo "クリーンアップ中..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GO) clean
	@echo "クリーンアップ完了"

# 依存パッケージのダウンロード
.PHONY: deps
deps:
	@echo "依存パッケージのダウンロード中..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "依存パッケージのダウンロード完了"

# 静的解析
.PHONY: lint
lint:
	@echo "静的解析中..."
	@which golangci-lint > /dev/null || (echo "golangci-lint がインストールされていません" && exit 1)
	golangci-lint run ./...
	@echo "静的解析完了"

# フォーマット
.PHONY: fmt
fmt:
	@echo "コードフォーマット中..."
	$(GO) fmt ./...
	@echo "フォーマット完了"

# 全体のチェック（フォーマット、静的解析、テスト）
.PHONY: check
check: fmt lint test
	@echo "チェック完了"

# ヘルプ
.PHONY: help
help:
	@echo "利用可能なターゲット:"
	@echo "  make build     - 本番用ビルド"
	@echo "  make dev       - 開発用ビルド（高速）"
	@echo "  make test      - テスト実行"
	@echo "  make coverage  - テストカバレッジレポート生成"
	@echo "  make install   - インストール"
	@echo "  make clean     - クリーンアップ"
	@echo "  make deps      - 依存パッケージのダウンロード"
	@echo "  make lint      - 静的解析"
	@echo "  make fmt       - コードフォーマット"
	@echo "  make check     - 全体チェック"
