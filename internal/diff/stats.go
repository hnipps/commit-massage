package diff

import (
	"fmt"
	"strings"
)

// Stats generates a git diff --stat style summary from a raw unified diff.
func Stats(rawDiff string) string {
	if rawDiff == "" {
		return ""
	}

	sections := parseSections(rawDiff)
	if len(sections) == 0 {
		return ""
	}

	type fileStat struct {
		path       string
		insertions int
		deletions  int
	}

	var stats []fileStat
	maxPathLen := 0
	totalInsertions := 0
	totalDeletions := 0

	for _, s := range sections {
		ins, del := countChanges(s.content)
		path := s.path
		if path == "" {
			continue
		}
		stats = append(stats, fileStat{path: path, insertions: ins, deletions: del})
		if len(path) > maxPathLen {
			maxPathLen = len(path)
		}
		totalInsertions += ins
		totalDeletions += del
	}

	if len(stats) == 0 {
		return ""
	}

	var b strings.Builder
	for _, fs := range stats {
		total := fs.insertions + fs.deletions
		bar := strings.Repeat("+", fs.insertions) + strings.Repeat("-", fs.deletions)
		// Cap the bar at 50 chars for readability.
		if len(bar) > 50 {
			ratio := float64(fs.insertions) / float64(total)
			plusCount := int(ratio*50 + 0.5)
			if plusCount > 50 {
				plusCount = 50
			}
			minusCount := 50 - plusCount
			bar = strings.Repeat("+", plusCount) + strings.Repeat("-", minusCount)
		}
		fmt.Fprintf(&b, " %-*s | %d %s\n", maxPathLen, fs.path, total, bar)
	}

	filesWord := "files"
	if len(stats) == 1 {
		filesWord = "file"
	}
	fmt.Fprintf(&b, " %d %s changed, %d insertions(+), %d deletions(-)", len(stats), filesWord, totalInsertions, totalDeletions)

	return b.String()
}

// countChanges counts insertions and deletions in a diff section.
func countChanges(content string) (insertions, deletions int) {
	for _, line := range strings.Split(content, "\n") {
		if len(line) == 0 {
			continue
		}
		// Only count lines within hunks (after @@ markers).
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			insertions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}
	return
}
