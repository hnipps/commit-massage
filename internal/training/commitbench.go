package training

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// Entry represents a single CommitBench dataset entry.
type Entry struct {
	Diff    string `json:"diff"`
	Message string `json:"message"`
}

// rawEntry supports common CommitBench field aliases.
type rawEntry struct {
	Diff          string `json:"diff"`
	Patch         string `json:"patch"`
	Message       string `json:"message"`
	Subject       string `json:"subject"`
	CommitMessage string `json:"commit_message"`
}

func (r rawEntry) toEntry() Entry {
	diff := r.Diff
	if diff == "" {
		diff = r.Patch
	}
	msg := r.Message
	if msg == "" {
		msg = r.Subject
	}
	if msg == "" {
		msg = r.CommitMessage
	}
	return Entry{Diff: diff, Message: msg}
}

// ReadEntries streams JSONL entries from r, calling fn for each valid entry.
// Entries missing both diff and message fields are skipped.
func ReadEntries(r io.Reader, fn func(Entry) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // up to 10MB per line
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw rawEntry
		if err := json.Unmarshal(line, &raw); err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
		entry := raw.toEntry()
		if entry.Diff == "" || entry.Message == "" {
			continue
		}
		if err := fn(entry); err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	return scanner.Err()
}
