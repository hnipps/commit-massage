# commit-massage

AI-generated conventional commit messages using a local LLM via an OpenAI-compatible API.

Installs a `prepare-commit-msg` git hook that automatically generates a commit message from your staged changes. Runs entirely locally — no cloud APIs, no latency.

## Prerequisites

- [Go](https://go.dev) 1.25+
- A local OpenAI-compatible API server (e.g. [LM Studio](https://lmstudio.ai)) running at `http://127.0.0.1:1234`
- A loaded model (default: `google/gemma-3n-e4b`)

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
| `COMMIT_MASSAGE_MODEL` | `google/gemma-3n-e4b` | Model to use |
| `COMMIT_MASSAGE_URL` | `http://127.0.0.1:1234` | OpenAI-compatible API server URL |

## Training Data Preparation

If you're fine-tuning your own model for commit message generation, the `prepare-training` command converts [CommitBench](https://huggingface.co/datasets/Maxscha/commitbench) JSONL into training-ready data that matches the exact prompt format used at inference time:

```sh
commit-massage prepare-training commitbench.jsonl training.jsonl
```

This applies the same diff preprocessing used during inference (noise filtering, file importance ranking, smart truncation), derives file change stats, and outputs OpenAI chat completion JSONL. Entries with only noise (lock files, generated code, etc.) are automatically skipped. Progress stats are reported to stderr.

See [docs/guides/mlx-fine-tuning.md](docs/guides/mlx-fine-tuning.md) for a complete guide on fine-tuning with MLX on Apple Silicon.

## Commands

| Command | Description |
|---|---|
| `commit-massage install [--force]` | Install the git hook |
| `commit-massage uninstall` | Remove the git hook |
| `commit-massage generate <file> [source]` | Generate a commit message (called by the hook) |
| `commit-massage prepare-training <in> <out>` | Prepare CommitBench JSONL for fine-tuning |
