# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

commit-massage is a Go CLI tool that installs a `prepare-commit-msg` git hook to auto-generate conventional commit messages using a local LLM via an OpenAI-compatible API (e.g. LM Studio). The hook intercepts commits, sends the staged diff to the LLM, and prepends a generated commit message.

## Commands

```bash
go build -o commit-massage .   # Build the binary
go run . install [--force]     # Install the git hook
go run . uninstall             # Remove the git hook
go vet ./...                   # Lint
go test ./...                  # Run all tests
go test ./internal/hook/       # Run tests for a single package
```

## Architecture

- `main.go` — CLI entrypoint; dispatches `install` / `uninstall` subcommands
- `internal/hook/` — Installs/uninstalls the `prepare-commit-msg` shell script into the repo's git hooks directory. Uses a `marker` comment to identify hooks it owns (safe uninstall).
- `internal/llm/` — HTTP client for OpenAI-compatible chat completions API (`/v1/chat/completions`).
- `internal/generate/` — Orchestrates commit message generation: gets the diff, calls the LLM, prepends the result to the commit message file.
- `internal/prompt/` — Contains the system prompt constant (`prompt.Text`) sent to the LLM.

The generated hook calls `commit-massage generate` at commit time. It truncates diffs over 20k chars and skips commits that already have a user-provided message.

## Key Details

- No external Go dependencies (stdlib only, uses `go 1.25.1`)
- The binary name is `commit-massage` (listed in `.gitignore`)
- Hook identification relies on a string marker (`"commit-massage"`) embedded in the script comment
- Default LLM server: `http://127.0.0.1:1234` (configurable via `COMMIT_MASSAGE_URL`)
- Default model: `google/gemma-3n-e4b` (configurable via `COMMIT_MASSAGE_MODEL`)
