# commit-massage

A CLI tool that installs a git `prepare-commit-msg` hook to automatically generate [conventional commit](https://www.conventionalcommits.org/) messages using [Gemini CLI](https://github.com/google-gemini/gemini-cli).

When you run `git commit`, the hook sends your staged diff to Gemini and prepopulates the commit message for you. If you provide your own message (`-m`), the hook stays out of the way.

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Gemini CLI](https://github.com/google-gemini/gemini-cli) installed and authenticated (`gemini` available on your `PATH`)
- A git repository

## Install

```sh
go install github.com/nicholls-inc/commit-massage@latest
```

Or build from source:

```sh
git clone https://github.com/nicholls-inc/commit-massage.git
cd commit-massage
go build -o commit-massage .
```

## Usage

### Install the hook

```sh
commit-massage install
```

This writes a `prepare-commit-msg` hook into your repo's `.git/hooks/` directory. If a hook already exists, use `--force` to overwrite it:

```sh
commit-massage install --force
```

### Uninstall the hook

```sh
commit-massage uninstall
```

Only removes hooks that were installed by commit-massage.

### Committing

Just commit as usual:

```sh
git add .
git commit
```

The hook will generate a conventional commit message from your staged changes and prepopulate your editor. Edit or accept it as you like.

Skipped automatically when you use `git commit -m`, merge commits, or squash commits.

## How it works

1. The `prepare-commit-msg` hook captures your staged diff (`git diff --cached`)
2. Large diffs are truncated to ~20,000 characters
3. The diff is piped to `gemini` with a prompt requesting a conventional commit message
4. The generated message is prepended to your commit message file

## Commit format

Generated messages follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): description
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

## License

MIT
