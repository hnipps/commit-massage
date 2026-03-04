# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

commit-massage is a Go CLI tool that installs a `prepare-commit-msg` git hook to auto-generate conventional commit messages using Gemini CLI. The hook intercepts commits, sends the staged diff to Gemini, and prepends a generated commit message.

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
- `internal/prompt/` — Contains the system prompt constant (`prompt.Text`) sent to Gemini. Embedded directly into the generated shell hook script.

The generated hook is a self-contained shell script that pipes `git diff --cached` to `gemini -p` at commit time. It truncates diffs over 20k chars and skips commits that already have a user-provided message.

## Key Details

- No external Go dependencies (stdlib only, uses `go 1.25.1`)
- The binary name is `commit-massage` (listed in `.gitignore`)
- Hook identification relies on a string marker (`"commit-massage"`) embedded in the script comment
