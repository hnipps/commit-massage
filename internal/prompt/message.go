package prompt

// BuildUserMessage constructs the user message sent to the LLM for commit
// message generation. It combines optional recent commit history, file stats,
// and the processed diff into the format expected by the system prompt.
func BuildUserMessage(recentCommits, fileStats, diff string) string {
	var msg string
	if recentCommits != "" {
		msg = "Recent commits (for style reference):\n" + recentCommits + "\n\n"
	}
	msg += "Files changed:\n" + fileStats + "\n\nDiff:\n" + diff
	return msg
}
