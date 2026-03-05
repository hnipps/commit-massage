package training

import (
	"encoding/json"

	"github.com/nicholls-inc/commit-massage/internal/llm"
	"github.com/nicholls-inc/commit-massage/internal/prompt"
)

type chatCompletion struct {
	Messages []llm.Message `json:"messages"`
}

// FormatChatCompletion formats a user message and commit message into an
// OpenAI chat completion JSONL line.
func FormatChatCompletion(userMessage, commitMessage string) ([]byte, error) {
	record := chatCompletion{
		Messages: []llm.Message{
			{Role: "system", Content: prompt.Text},
			{Role: "user", Content: userMessage},
			{Role: "assistant", Content: commitMessage},
		},
	}
	return json.Marshal(record)
}
