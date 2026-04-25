package articletext

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const DefaultTimeout = 45 * time.Second

type Extractor struct {
	Command string
	Timeout time.Duration
}

func NewExtractor() Extractor {
	return Extractor{Command: "trafilatura", Timeout: DefaultTimeout}
}

func (e Extractor) ExtractURL(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("URL is required")
	}
	command := e.Command
	if command == "" {
		command = "trafilatura"
	}
	limit := e.Timeout
	if limit <= 0 {
		limit = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, CLIArgs(rawURL)...)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("article extraction timed out")
	}
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("article extraction failed: %s", msg)
	}
	article := strings.TrimSpace(string(out))
	if article == "" {
		return "", errors.New("trafilatura did not find readable article content")
	}
	return article, nil
}

func CLIArgs(rawURL string) []string {
	return []string{"--markdown", "--images", "--no-comments", "--no-tables", "-u", rawURL}
}
