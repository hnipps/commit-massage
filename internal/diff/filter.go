package diff

import (
	"path/filepath"
	"strings"
)

// lockFileNames lists exact file names that are considered lock files.
var lockFileNames = map[string]bool{
	"go.sum":            true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":   true,
	"Cargo.lock":        true,
	"Gemfile.lock":      true,
	"poetry.lock":       true,
	"composer.lock":     true,
}

// lockFileExts lists extensions that indicate lock files.
var lockFileExts = map[string]bool{
	".lock": true,
}

// isLockFile reports whether the file path refers to a lock file.
func isLockFile(path string) bool {
	base := filepath.Base(path)
	if lockFileNames[base] {
		return true
	}
	if lockFileExts[filepath.Ext(base)] {
		return true
	}
	// *-lock.json pattern
	if strings.HasSuffix(base, "-lock.json") {
		return true
	}
	return false
}

// isGeneratedCode reports whether the section represents generated code,
// based on the file path or content of the first few lines.
func isGeneratedCode(path, content string) bool {
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, "_gen.go") ||
		strings.Contains(base, ".generated.") {
		return true
	}
	// Check first few lines of the diff content for generation markers.
	lines := strings.SplitN(content, "\n", 20)
	for _, line := range lines {
		if strings.Contains(line, "Code generated") || strings.Contains(line, "DO NOT EDIT") {
			return true
		}
	}
	return false
}

// isBinaryFile reports whether the section describes a binary file change.
func isBinaryFile(content string) bool {
	return strings.Contains(content, "Binary files")
}

// isVendored reports whether the file is under a vendor directory.
func isVendored(path string) bool {
	return hasPathPrefix(path, "vendor/")
}

// filterSection checks a section and returns a replacement placeholder if the
// file should be filtered. If no filtering is needed it returns the original
// content unchanged and false.
func filterSection(path, content string) (string, bool) {
	base := filepath.Base(path)

	if isLockFile(path) {
		return "[lock file: " + base + " changed]", true
	}
	if isBinaryFile(content) {
		return "[binary file: " + base + " changed]", true
	}
	if isVendored(path) {
		return "[vendor file: " + base + " changed]", true
	}
	if isGeneratedCode(path, content) {
		return "[generated file: " + base + " changed]", true
	}
	return content, false
}
