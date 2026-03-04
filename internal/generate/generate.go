package generate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nicholls-inc/commit-massage/internal/log"
	"github.com/nicholls-inc/commit-massage/internal/llm"
	"github.com/nicholls-inc/commit-massage/internal/prompt"
)

const maxDiffLen = 20000

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

	if len(diff) > maxDiffLen {
		diff = diff[:maxDiffLen] + "\n[diff truncated]"
	}

	model := envOrDefault("COMMIT_MASSAGE_MODEL", "google/gemma-3n-e4b")
	baseURL := envOrDefault("COMMIT_MASSAGE_URL", "http://127.0.0.1:1234")

	userMessage := "Files changed:\n" + stat + "\n\nDiff:\n" + diff

	client := llm.NewClient(baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	spinner := log.Start("Generating commit message…")

	msg, err := client.Chat(ctx, model, []llm.Message{
		{Role: "system", Content: prompt.Text},
		{Role: "user", Content: userMessage},
	})
	if err != nil {
		spinner.Fail("Failed to generate commit message")
		return err
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
