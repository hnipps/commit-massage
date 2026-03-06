package training

import (
	"regexp"
	"strings"
)

var subjectRe = regexp.MustCompile(
	`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([a-z0-9._-]+\))?!?: [a-z]`,
)

// ValidateMessage checks whether msg follows the conventional commit rules
// from the system prompt. It returns "" if valid, or a short reason if not.
func ValidateMessage(msg string) string {
	lines := strings.SplitN(msg, "\n", 2)
	subject := lines[0]

	if !subjectRe.MatchString(subject) {
		return "subject-format"
	}
	if len(subject) > 72 {
		return "subject-too-long"
	}
	if strings.HasSuffix(subject, ".") {
		return "subject-trailing-period"
	}

	if len(lines) < 2 {
		return ""
	}

	body := lines[1]
	bodyLines := strings.Split(body, "\n")

	if len(bodyLines) == 0 || bodyLines[0] != "" {
		return "body-no-blank-line"
	}

	for _, line := range bodyLines[1:] {
		if len(line) > 72 {
			return "body-line-too-long"
		}
	}

	return ""
}
