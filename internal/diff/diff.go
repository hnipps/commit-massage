// Package diff provides preprocessing for unified diffs: filtering noise,
// ranking files by importance, and applying smart per-file truncation.
package diff

import (
	"sort"
	"strings"
)

// section represents one file's portion of a unified diff.
type section struct {
	path    string
	content string // full text including header
	tier    int
}

// Process filters noise, ranks files by importance, and applies smart
// per-file truncation to fit within maxLen characters.
func Process(rawDiff string, maxLen int) string {
	if rawDiff == "" {
		return ""
	}

	sections := parseSections(rawDiff)
	if len(sections) == 0 {
		return rawDiff
	}

	// Apply noise filters.
	for i := range sections {
		replacement, filtered := filterSection(sections[i].path, sections[i].content)
		if filtered {
			sections[i].content = replacement
		}
	}

	// Assign tiers.
	for i := range sections {
		sections[i].tier = classifyTier(sections[i].path)
	}

	// Check if we're within budget.
	if totalLen(sections) <= maxLen {
		return joinSections(sections)
	}

	// Truncate starting from lowest-priority (highest tier number).
	truncate(sections, maxLen)

	return joinSections(sections)
}

// parseSections splits a raw unified diff into per-file sections.
func parseSections(raw string) []section {
	const marker = "diff --git "
	parts := strings.Split(raw, marker)

	var sections []section
	for i, part := range parts {
		if i == 0 {
			// Text before the first "diff --git" marker (usually empty).
			if strings.TrimSpace(part) != "" {
				sections = append(sections, section{content: part})
			}
			continue
		}
		content := marker + part
		path := extractPath(content)
		sections = append(sections, section{
			path:    path,
			content: content,
		})
	}
	return sections
}

// extractPath pulls the file path from a "diff --git a/X b/X" header line.
func extractPath(header string) string {
	// The header looks like: diff --git a/path b/path
	first := strings.SplitN(header, "\n", 2)[0]
	parts := strings.SplitN(first, " b/", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	// Fallback: try to parse from a/ prefix.
	parts = strings.SplitN(first, " a/", 2)
	if len(parts) == 2 {
		tok := strings.SplitN(parts[1], " ", 2)
		return tok[0]
	}
	return ""
}

// truncate reduces section content starting from the lowest-priority files
// until total length fits within maxLen.
func truncate(sections []section, maxLen int) {
	// Build an index sorted by tier descending (lowest priority first),
	// then by content length descending within the same tier.
	indices := make([]int, len(sections))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(a, b int) bool {
		ia, ib := indices[a], indices[b]
		if sections[ia].tier != sections[ib].tier {
			return sections[ia].tier > sections[ib].tier
		}
		return len(sections[ia].content) > len(sections[ib].content)
	})

	total := totalLen(sections)

	for _, idx := range indices {
		if total <= maxLen {
			return
		}

		s := &sections[idx]
		oldLen := len(s.content)

		// Budget for this section: total budget minus everything else.
		budget := maxLen - (total - oldLen)

		// First attempt: truncate hunks, keep header + first N lines.
		truncated := truncateHunks(s.content, budget)
		if truncated != s.content {
			s.content = truncated
			total += len(s.content) - oldLen
			if total <= maxLen {
				return
			}
			oldLen = len(s.content)
		}

		// Second attempt: replace with a one-line placeholder.
		name := s.path
		if name == "" {
			name = "file"
		}
		s.content = "[" + name + ": diff omitted]"
		total += len(s.content) - oldLen
	}
}

// truncateHunks keeps the diff header and up to budget characters of hunk
// content, appending a truncation marker.
func truncateHunks(content string, budget int) string {
	if budget <= 0 {
		return content // will be replaced by placeholder in caller
	}
	if len(content) <= budget {
		return content
	}

	// Keep as much as the budget allows.
	truncated := content[:budget]
	// Try to cut at a newline boundary for cleanliness.
	if idx := strings.LastIndex(truncated, "\n"); idx > 0 {
		truncated = truncated[:idx]
	}
	return truncated + "\n[... truncated]"
}

func totalLen(sections []section) int {
	n := 0
	for i, s := range sections {
		n += len(s.content)
		if i > 0 {
			n++ // newline separator
		}
	}
	return n
}

func joinSections(sections []section) string {
	parts := make([]string, len(sections))
	for i, s := range sections {
		parts[i] = s.content
	}
	return strings.Join(parts, "\n")
}
