package training

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nicholls-inc/commit-massage/internal/prompt"
)

func TestProcess(t *testing.T) {
	cleanDiff := "diff --git a/main.go b/main.go\n" +
		"index abc..def 100644\n" +
		"--- a/main.go\n" +
		"+++ b/main.go\n" +
		"@@ -1,3 +1,4 @@\n" +
		" package main\n" +
		"+import \"fmt\"\n" +
		"+func hello() { fmt.Println(\"hi\") }\n" +
		" \n"

	lockOnlyDiff := "diff --git a/go.sum b/go.sum\n" +
		"index abc..def 100644\n" +
		"--- a/go.sum\n" +
		"+++ b/go.sum\n" +
		"@@ -1,3 +1,5 @@\n" +
		"+hash1\n" +
		"+hash2\n"

	tests := []struct {
		name        string
		input       string
		wantWritten int
		wantSkipped int
		checkOutput func(t *testing.T, output string)
	}{
		{
			name: "clean diff produces output",
			input: jsonLine(t, map[string]string{
				"diff":    cleanDiff,
				"message": "feat: add hello function",
			}),
			wantWritten: 1,
			wantSkipped: 0,
			checkOutput: func(t *testing.T, output string) {
				lines := nonEmptyLines(output)
				if len(lines) != 1 {
					t.Fatalf("expected 1 line, got %d", len(lines))
				}
				var record struct {
					Messages []struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					} `json:"messages"`
				}
				if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				if len(record.Messages) != 3 {
					t.Fatalf("expected 3 messages, got %d", len(record.Messages))
				}
				if record.Messages[0].Role != "system" || record.Messages[0].Content != prompt.Text {
					t.Error("system message mismatch")
				}
				if record.Messages[1].Role != "user" {
					t.Error("expected user role")
				}
				if !strings.Contains(record.Messages[1].Content, "Files changed:") {
					t.Error("user message missing file stats")
				}
				if !strings.Contains(record.Messages[1].Content, "Diff:") {
					t.Error("user message missing diff")
				}
				if record.Messages[2].Role != "assistant" || record.Messages[2].Content != "feat: add hello function" {
					t.Error("assistant message mismatch")
				}
			},
		},
		{
			name: "lock-file-only diff is skipped",
			input: jsonLine(t, map[string]string{
				"diff":    lockOnlyDiff,
				"message": "chore: update deps",
			}),
			wantWritten: 0,
			wantSkipped: 1,
		},
		{
			name: "field aliases work",
			input: jsonLine(t, map[string]string{
				"patch":   cleanDiff,
				"subject": "feat: aliased fields",
			}),
			wantWritten: 1,
			wantSkipped: 0,
		},
		{
			name: "entries missing fields are skipped",
			input: jsonLine(t, map[string]string{
				"diff": cleanDiff,
			}),
			wantWritten: 0,
			wantSkipped: 0, // skipped in ReadEntries, not counted
		},
		{
			name: "oversized diff gets truncated",
			input: jsonLine(t, map[string]string{
				"diff":    makeLargeDiff(25000),
				"message": "feat: big change",
			}),
			wantWritten: 1,
			wantSkipped: 0,
			checkOutput: func(t *testing.T, output string) {
				lines := nonEmptyLines(output)
				if len(lines) != 1 {
					t.Fatalf("expected 1 line, got %d", len(lines))
				}
				var record struct {
					Messages []struct {
						Content string `json:"content"`
					} `json:"messages"`
				}
				if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}
				// The user message (which contains the diff) should be
				// under budget after processing.
				userContent := record.Messages[1].Content
				if len(userContent) > 25000 {
					t.Errorf("user message too large: %d chars", len(userContent))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			stats, err := process(strings.NewReader(tt.input), &out)
			if err != nil {
				t.Fatalf("process error: %v", err)
			}
			if stats.Written != tt.wantWritten {
				t.Errorf("written: got %d, want %d", stats.Written, tt.wantWritten)
			}
			if stats.Skipped != tt.wantSkipped {
				t.Errorf("skipped: got %d, want %d", stats.Skipped, tt.wantSkipped)
			}
			if tt.checkOutput != nil {
				tt.checkOutput(t, out.String())
			}
		})
	}
}

func jsonLine(t *testing.T, m map[string]string) string {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func makeLargeDiff(size int) string {
	var b strings.Builder
	b.WriteString("diff --git a/big.go b/big.go\n")
	b.WriteString("--- a/big.go\n")
	b.WriteString("+++ b/big.go\n")
	b.WriteString("@@ -1,1 +1,1000 @@\n")
	for b.Len() < size {
		b.WriteString("+// padding line to make the diff large enough for truncation testing\n")
	}
	return b.String()
}
