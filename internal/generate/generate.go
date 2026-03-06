package generate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	diffpkg "github.com/nicholls-inc/commit-massage/internal/diff"
	"github.com/nicholls-inc/commit-massage/internal/llm"
	"github.com/nicholls-inc/commit-massage/internal/log"
	"github.com/nicholls-inc/commit-massage/internal/prompt"
)

var maxDiffLen = diffpkg.MaxLen
const defaultTimeout  = 30 // seconds

// Run generates a commit message and prepends it to msgFile.
// source is the second argument passed to prepare-commit-msg by git.
func Run(msgFile, source string) error {
	switch source {
	case "message", "merge", "squash":
		return nil
	}

	diff, err := gitOutput("diff", "--cached", "--no-color", "--histogram")
	if err != nil {
		return fmt.Errorf("git diff: %w", err)
	}
	if len(diff) == 0 {
		return nil
	}

	stat, err := gitOutput("diff", "--cached", "--stat", "--no-color")
	if err != nil {
		return fmt.Errorf("git diff --stat: %w", err)
	}

	diff = diffpkg.Process(diff, maxDiffLen)

	// Fetch recent commit history for style context; silently skip on error
	// (e.g. first commit in repo).
	recentLog, _ := gitOutput("log", "--oneline", "-10")

	model := envOrDefault("COMMIT_MASSAGE_MODEL", "google/gemma-3n-e4b")
	baseURL := envOrDefault("COMMIT_MASSAGE_URL", "http://127.0.0.1:1234")

	userMessage := prompt.BuildUserMessage(recentLog, stat, diff)

	client := llm.NewClient(baseURL)

	timeout := defaultTimeout
	if v := os.Getenv("COMMIT_MASSAGE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = n
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	spinner := log.Start("Generating commit message…")

	msg, err := client.Chat(ctx, model, []llm.Message{
		{Role: "system", Content: prompt.Text},
		{Role: "user", Content: userMessage},
	})
	if err != nil {
		spinner.Fail("Failed to generate commit message")
		fmt.Fprintf(os.Stderr, "commit-massage: %s\n", err)
		return nil
	}

	spinner.Stop("Commit message generated")

	existing, err := os.ReadFile(msgFile)
	if err != nil {
		return fmt.Errorf("read commit message file: %w", err)
	}

	content := msg + "\n" + string(existing)
	if err := os.WriteFile(msgFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write commit message file: %w", err)
	}

	return nil
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
