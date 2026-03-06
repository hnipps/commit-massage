package prompt

import (
	"strings"
	"testing"
)

func TestBuildUserMessage(t *testing.T) {
	tests := []struct {
		name          string
		recentCommits string
		fileStats     string
		diff          string
		wantContains  []string
		wantMissing   []string
	}{
		{
			name:          "with recent commits",
			recentCommits: "abc123 feat: add foo",
			fileStats:     " main.go | 5 ++--",
			diff:          "diff --git a/main.go b/main.go",
			wantContains:  []string{"Recent commits (for style reference):", "abc123 feat: add foo", "Files changed:", "Diff:"},
		},
		{
			name:         "without recent commits",
			fileStats:    " main.go | 5 ++--",
			diff:         "diff --git a/main.go b/main.go",
			wantContains: []string{"Files changed:", "Diff:"},
			wantMissing:  []string{"Recent commits"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildUserMessage(tt.recentCommits, tt.fileStats, tt.diff)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("result missing %q:\n%s", want, got)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(got, missing) {
					t.Errorf("result should not contain %q:\n%s", missing, got)
				}
			}
		})
	}
}
