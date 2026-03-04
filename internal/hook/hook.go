package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nicholls-inc/commit-massage/internal/prompt"
)

const marker = "commit-massage"

var scriptTemplate = `#!/bin/sh
# ` + marker + `: AI-generated conventional commits via Gemini CLI

MSG_FILE="$1"
SOURCE="$2"

# Skip when user provides their own message or git generates one
case "$SOURCE" in
  message|merge|squash) exit 0 ;;
esac

DIFF=$(git diff --cached)
[ -z "$DIFF" ] && exit 0

STAT=$(git diff --cached --stat)

# Truncate large diffs (keep first ~20000 chars)
if [ ${#DIFF} -gt 20000 ]; then
  DIFF=$(printf '%s' "$DIFF" | head -c 20000)
  DIFF="${DIFF}
[diff truncated]"
fi

PROMPT='` + prompt.Text + `

Files changed:
'"$STAT"'

Diff:
'"$DIFF"

MSG=$(printf '%s' "$PROMPT" | gemini -p "Generate a commit message based on the instructions and diff provided via stdin." 2>/dev/null)

if [ -n "$MSG" ]; then
  EXISTING=$(cat "$MSG_FILE")
  printf '%s\n%s' "$MSG" "$EXISTING" > "$MSG_FILE"
fi
`

const hookName = "prepare-commit-msg"

func hooksDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--git-path", "hooks").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository (or git not installed)")
	}

	dir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(dir) {
		toplevel, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return "", fmt.Errorf("could not determine repository root")
		}
		dir = filepath.Join(strings.TrimSpace(string(toplevel)), dir)
	}

	return dir, nil
}

// Install writes the prepare-commit-msg hook script. If force is false,
// it refuses to overwrite an existing hook file.
func Install(force bool) error {
	dir, err := hooksDir()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(dir, hookName)

	if !force {
		if _, err := os.Stat(hookPath); err == nil {
			return fmt.Errorf("hook already exists at %s (use --force to overwrite)", hookPath)
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create hooks directory: %w", err)
	}

	if err := os.WriteFile(hookPath, []byte(scriptTemplate), 0755); err != nil {
		return fmt.Errorf("could not write hook: %w", err)
	}

	fmt.Printf("Installed prepare-commit-msg hook at %s\n", hookPath)
	return nil
}

// Uninstall removes the prepare-commit-msg hook, but only if it was
// installed by commit-massage (identified by the marker comment).
func Uninstall() error {
	dir, err := hooksDir()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(dir, hookName)

	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no prepare-commit-msg hook found")
		}
		return fmt.Errorf("could not read hook: %w", err)
	}

	if !strings.Contains(string(data), marker) {
		return fmt.Errorf("hook at %s was not installed by commit-massage, refusing to remove", hookPath)
	}

	if err := os.Remove(hookPath); err != nil {
		return fmt.Errorf("could not remove hook: %w", err)
	}

	fmt.Printf("Removed prepare-commit-msg hook from %s\n", hookPath)
	return nil
}
