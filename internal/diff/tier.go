package diff

import (
	"path/filepath"
	"strings"
)

// Tier levels: lower number = higher priority.
const (
	tierSource  = 1
	tierConfig  = 2
	tierTest    = 3
	tierDocs    = 4
	tierStyle   = 5
)

// sourceExts lists file extensions classified as tier-1 source code.
var sourceExts = map[string]bool{
	".go": true, ".ts": true, ".js": true, ".tsx": true, ".jsx": true,
	".py": true, ".rs": true, ".java": true, ".rb": true,
	".c": true, ".cpp": true, ".h": true, ".cs": true,
	".swift": true, ".kt": true,
}

// docExts lists file extensions classified as tier-4 documentation.
var docExts = map[string]bool{
	".md": true, ".txt": true, ".rst": true, ".adoc": true,
}

// classifyTier returns the tier for a given file path.
func classifyTier(path string) int {
	base := filepath.Base(path)
	ext := filepath.Ext(path)

	// Tier 3: tests (checked before source so _test.go wins over .go)
	if strings.HasSuffix(base, "_test.go") {
		return tierTest
	}
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return tierTest
	}
	if hasPathPrefix(path, "test/") || hasPathPrefix(path, "tests/") {
		return tierTest
	}

	// Tier 1: source code
	if sourceExts[ext] {
		return tierSource
	}

	// Tier 2: behavioural config
	if isBehavioralConfig(path, base) {
		return tierConfig
	}

	// Tier 5: style/formatting config
	if isStyleConfig(base) {
		return tierStyle
	}

	// Tier 4: documentation
	if docExts[ext] {
		return tierDocs
	}

	// Default to source-level priority for unknown files.
	return tierSource
}

func isBehavioralConfig(path, base string) bool {
	switch base {
	case "Dockerfile", "go.mod", "Makefile", "Cargo.toml",
		"package.json", "tsconfig.json":
		return true
	}
	if strings.HasPrefix(base, "docker-compose") {
		return true
	}
	if hasPathPrefix(path, ".github/workflows/") {
		return true
	}
	return false
}

func isStyleConfig(base string) bool {
	switch base {
	case ".editorconfig", ".gitignore":
		return true
	}
	if strings.HasPrefix(base, ".eslintrc") || strings.HasPrefix(base, ".prettierrc") {
		return true
	}
	return false
}

// hasPathPrefix checks whether path starts with the given directory prefix
// using forward slashes (as used in diff --git headers).
func hasPathPrefix(path, prefix string) bool {
	// Normalise to forward slashes for consistent matching.
	p := filepath.ToSlash(path)
	return strings.HasPrefix(p, prefix) || strings.Contains(p, "/"+prefix)
}
