package training

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nicholls-inc/commit-massage/internal/diff"
	"github.com/nicholls-inc/commit-massage/internal/prompt"
)

var maxDiffLen = diff.MaxLen

// Stats tracks pipeline processing statistics.
type Stats struct {
	Total       int
	Written     int
	Skipped     int
	SkipReasons map[string]int
}

// Run reads CommitBench JSONL from inputPath, processes each entry through
// the diff pipeline, and writes OpenAI chat completion JSONL to outputPath.
// Statistics are reported to stderr.
func Run(inputPath, outputPath string) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	stats, err := process(inFile, outFile)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Done: %d total, %d written, %d skipped\n",
		stats.Total, stats.Written, stats.Skipped)
	if len(stats.SkipReasons) > 0 {
		fmt.Fprintf(os.Stderr, "Skip reasons:\n")
		for reason, count := range stats.SkipReasons {
			fmt.Fprintf(os.Stderr, "  %-30s %d\n", reason, count)
		}
	}
	return nil
}

func process(r io.Reader, w io.Writer) (Stats, error) {
	stats := Stats{SkipReasons: make(map[string]int)}
	bw := bufio.NewWriter(w)

	err := ReadEntries(r, func(entry Entry) error {
		stats.Total++

		entry.Message = CleanMessage(entry.Message)

		if reason := ValidateMessage(entry.Message); reason != "" {
			stats.Skipped++
			stats.SkipReasons[reason]++
			return nil
		}

		processed := diff.Process(entry.Diff, maxDiffLen)
		if processed == "" || isAllPlaceholders(processed) {
			stats.Skipped++
			stats.SkipReasons["diff-empty"]++
			return nil
		}

		fileStats := diff.Stats(entry.Diff)
		userMessage := prompt.BuildUserMessage("", fileStats, processed)

		line, err := FormatChatCompletion(userMessage, entry.Message)
		if err != nil {
			return fmt.Errorf("format: %w", err)
		}

		if _, err := bw.Write(line); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		if err := bw.WriteByte('\n'); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}

		stats.Written++
		return nil
	})
	if err != nil {
		return stats, err
	}

	if err := bw.Flush(); err != nil {
		return stats, fmt.Errorf("flush output: %w", err)
	}
	return stats, nil
}

// isAllPlaceholders returns true if the processed diff contains no real diff
// content — only noise-filter placeholders like "[lock file: X changed]".
func isAllPlaceholders(processed string) bool {
	for _, line := range strings.Split(processed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			return false
		}
	}
	return true
}
