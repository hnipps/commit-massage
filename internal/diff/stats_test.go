package diff

import (
	"strings"
	"testing"
)

func TestStats(t *testing.T) {
	tests := []struct {
		name         string
		rawDiff      string
		wantContains []string
	}{
		{
			name:    "empty diff",
			rawDiff: "",
		},
		{
			name: "single file",
			rawDiff: "diff --git a/main.go b/main.go\n" +
				"index abc..def 100644\n" +
				"--- a/main.go\n" +
				"+++ b/main.go\n" +
				"@@ -1,3 +1,5 @@\n" +
				" package main\n" +
				"+import \"fmt\"\n" +
				"+\n" +
				"+func hello() { fmt.Println(\"hi\") }\n" +
				" \n" +
				"-// old comment\n",
			wantContains: []string{"main.go", "4 +++", "1 file changed", "3 insertions(+)", "1 deletions(-)"},
		},
		{
			name: "multiple files",
			rawDiff: "diff --git a/a.go b/a.go\n" +
				"--- a/a.go\n" +
				"+++ b/a.go\n" +
				"@@ -1 +1,2 @@\n" +
				"+new line\n" +
				"diff --git a/b.go b/b.go\n" +
				"--- a/b.go\n" +
				"+++ b/b.go\n" +
				"@@ -1,2 +1 @@\n" +
				"-removed\n",
			wantContains: []string{"a.go", "b.go", "2 files changed", "1 insertions(+)", "1 deletions(-)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Stats(tt.rawDiff)
			if tt.rawDiff == "" {
				if got != "" {
					t.Errorf("expected empty, got %q", got)
				}
				return
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("result missing %q:\n%s", want, got)
				}
			}
		})
	}
}
