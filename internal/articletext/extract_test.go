package articletext

import (
	"strings"
	"testing"
)

func TestCLIArgsExtractMarkdownFromURL(t *testing.T) {
	args := CLIArgs("https://example.com/article")
	want := []string{"--markdown", "--images", "--no-comments", "--no-tables", "-u", "https://example.com/article"}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("CLIArgs() = %#v, want %#v", args, want)
	}
}

func TestExtractURLRejectsEmptyURL(t *testing.T) {
	_, err := NewExtractor().ExtractURL(t.Context(), " ")
	if err == nil {
		t.Fatal("expected error")
	}
}
