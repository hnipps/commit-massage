# commit-massage

AI-generated conventional commit messages using a local LLM via [Ollama](https://ollama.com).

Installs a `prepare-commit-msg` git hook that automatically generates a commit message from your staged changes. Runs entirely locally — no cloud APIs, no latency.

## Prerequisites

- [Go](https://go.dev) 1.25+
- [Ollama](https://ollama.com) installed and running
- A pulled model (default: `gemma3:1b`)

```sh
ollama serve        # start the server (if not already running)
ollama pull gemma3:1b # pull the default model
```

## Install

```sh
go install github.com/nicholls-inc/commit-massage@latest
```

Then install the git hook in any repository:

```sh
cd your-repo
commit-massage install
```

Use `--force` to overwrite an existing `prepare-commit-msg` hook.

## Usage

Just commit as normal:

```sh
git add .
git commit
```

The hook generates a conventional commit message from your staged diff and pre-fills the editor. Edit or accept it.

Messages you provide explicitly are left alone:

```sh
git commit -m "my manual message"  # hook does nothing
```

## Uninstall

```sh
commit-massage uninstall
```

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `COMMIT_MASSAGE_MODEL` | `gemma3:1b` | Ollama model to use |
| `COMMIT_MASSAGE_OLLAMA_URL` | `http://localhost:11434` | Ollama server URL |

## Commands

| Command | Description |
|---|---|
| `commit-massage install [--force]` | Install the git hook |
| `commit-massage uninstall` | Remove the git hook |
| `commit-massage generate <file> [source]` | Generate a commit message (called by the hook) |
