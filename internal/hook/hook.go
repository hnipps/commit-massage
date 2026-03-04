package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nicholls-inc/commit-massage/internal/log"
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

# Spinner while generating
spin() {
  set -- "⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏"
  while :; do
    for f; do printf '\r  %s Generating commit message...' "$f"; sleep 0.08; done
  done
}
spin &
SPIN_PID=$!
trap 'kill $SPIN_PID 2>/dev/null' EXIT

MSG=$(printf '%s' "$PROMPT" | gemini -p "Generate a commit message based on the instructions and diff provided via stdin." 2>/dev/null)

kill $SPIN_PID 2>/dev/null
trap - EXIT
wait $SPIN_PID 2>/dev/null

if [ -n "$MSG" ]; then
  printf '\r\033[K  ✓ Generated commit message\n'
  EXISTING=$(cat "$MSG_FILE")
  printf '%s\n%s' "$MSG" "$EXISTING" > "$MSG_FILE"
else
  printf '\r\033[K  ✗ Failed to generate commit message\n'
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
	s := log.Start("Locating git hooks directory...")
	dir, err := hooksDir()
	if err != nil {
		s.Fail("Not a git repository")
		return err
	}
	s.Stop("Found hooks directory")

	hookPath := filepath.Join(dir, hookName)

	if !force {
		s = log.Start("Checking for existing hook...")
		if _, err := os.Stat(hookPath); err == nil {
			s.Fail("Hook already exists")
			return fmt.Errorf("hook already exists at %s (use --force to overwrite)", hookPath)
		}
		s.Stop("No conflicting hook found")
	}

	s = log.Start("Writing hook script...")
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.Fail("Could not create hooks directory")
		return fmt.Errorf("could not create hooks directory: %w", err)
	}

	if err := os.WriteFile(hookPath, []byte(scriptTemplate), 0755); err != nil {
		s.Fail("Could not write hook")
		return fmt.Errorf("could not write hook: %w", err)
	}
	s.Stop(fmt.Sprintf("Installed hook at %s", hookPath))

	return nil
}

// Uninstall removes the prepare-commit-msg hook, but only if it was
// installed by commit-massage (identified by the marker comment).
func Uninstall() error {
	s := log.Start("Locating git hooks directory...")
	dir, err := hooksDir()
	if err != nil {
		s.Fail("Not a git repository")
		return err
	}
	s.Stop("Found hooks directory")

	hookPath := filepath.Join(dir, hookName)

	s = log.Start("Verifying hook ownership...")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Fail("No hook found")
			return fmt.Errorf("no prepare-commit-msg hook found")
		}
		s.Fail("Could not read hook")
		return fmt.Errorf("could not read hook: %w", err)
	}

	if !strings.Contains(string(data), marker) {
		s.Fail("Hook not owned by commit-massage")
		return fmt.Errorf("hook at %s was not installed by commit-massage, refusing to remove", hookPath)
	}
	s.Stop("Hook belongs to commit-massage")

	s = log.Start("Removing hook...")
	if err := os.Remove(hookPath); err != nil {
		s.Fail("Could not remove hook")
		return fmt.Errorf("could not remove hook: %w", err)
	}
	s.Stop(fmt.Sprintf("Removed hook from %s", hookPath))

	return nil
}
