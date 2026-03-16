# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

- GLM API integration for dynamic question generation
- `--llm-model` flag to override the LLM model
- `APIKeyError` type for user-friendly API key error messages
- `NewLLMClient` factory function for GLM client creation

### Changed

- **BREAKING**: LLM backend changed from Claude CLI to GLM API
- Environment variable `GLM_API_KEY` is now required for LLM features
- `spec-gather` command now uses GLM API instead of Claude CLI

### Removed

- **BREAKING**: Claude CLI integration removed (no longer required)
- `--claude-path` CLI flag removed
- Config field `llm.claude_path` is no longer used
- `NewClaudeClient` function and `ClaudeClient` type removed
- `WithCLIPath` and `WithCLIArgs` options removed

### Migration Guide

If you were using the Claude CLI integration:

1. Get a GLM API key from https://open.bigmodel.cn/
2. Set the environment variable:
   ```bash
   export GLM_API_KEY=your_api_key_here
   ```
3. Remove any `claude_path` settings from your `.ags.yaml` file
4. Replace `--claude-path` flag usage with `--llm-model` if you need to override the model

## [0.1.0] - Initial Release

### Added

- Initial release of agentic-shell
- `spec-gather` command for interactive specification gathering
- `generate` command for agent definition generation
- Dynamic question generation with LLM
- YAML and JSON output support
- Configuration file support (`.ags.yaml`)
