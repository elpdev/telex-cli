package emailtext

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func TestRenderPrefersPlainText(t *testing.T) {
	rendered, err := Render("Plain body", "<p>HTML body</p>", 80)
	if err != nil {
		t.Fatal(err)
	}
	rendered = stripANSI(rendered)
	if !strings.Contains(rendered, "Plain body") || strings.Contains(rendered, "HTML body") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestRenderConvertsHTMLWithInlineURLs(t *testing.T) {
	rendered, err := Render("", `<html><body><p>Hello <a href="https://example.com/read">Read more</a></p></body></html>`, 80)
	if err != nil {
		t.Fatal(err)
	}
	rendered = stripANSI(rendered)
	if !strings.Contains(rendered, "Hello") || !strings.Contains(rendered, "Read more (https://example.com/read)") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func stripANSI(value string) string {
	return ansiRE.ReplaceAllString(value, "")
}

func TestRenderCleansLongTrackingURLs(t *testing.T) {
	rendered, err := Render("", `<p><a href="https://open.substack.com/pub/unusualwhales/p/the-earnings-this-week-economic-events-0ce?utm_source=unread-posts-digest-email&amp;inbox=true&amp;utm_medium=email&amp;token=abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz">Read more</a></p>`, 100)
	if err != nil {
		t.Fatal(err)
	}
	rendered = stripANSI(rendered)
	if strings.Contains(rendered, "token=") || strings.Contains(rendered, "utm_") || strings.Contains(rendered, "inbox=") {
		t.Fatalf("rendered = %q", rendered)
	}
	if !strings.Contains(rendered, "Read more (https://open.substack.com/pub/unusualwhales/p/the-earnings-this-week-economic-events") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestLinksExtractsHTMLAndPlainURLs(t *testing.T) {
	links := Links("See https://example.net/plain.", `<p><a href="https://example.com/read?token=abc">Read more</a><a href="mailto:nope@example.com">email</a></p>`)
	if len(links) != 2 {
		t.Fatalf("links = %#v", links)
	}
	if links[0].Text != "Read more" || links[0].URL != "https://example.com/read?token=abc" {
		t.Fatalf("first link = %#v", links[0])
	}
	if links[1].URL != "https://example.net/plain" {
		t.Fatalf("second link = %#v", links[1])
	}
}

func TestRenderRejectsCSSHeavyPlainTextAndUsesHTML(t *testing.T) {
	plain := strings.Join([]string{
		"Email from Substack",
		"@media (max-width: 1024px) {",
		".typography .pullquote-align-left,",
		".typography.editor .pullquote-align-left {",
		"font-size: 14px;",
		"line-height: 20px;",
		"display: none;",
		"}",
		"@media screen and (max-width: 650px) {",
		".tweet .tweet-text {",
		"height: 200px;",
		"}",
	}, "\n")
	rendered, err := Render(plain, `<html><head><style>.bad { display: none; }</style></head><body><p>Actual email body</p></body></html>`, 80)
	if err != nil {
		t.Fatal(err)
	}
	rendered = stripANSI(rendered)
	if strings.Contains(rendered, "@media") || !strings.Contains(rendered, "Actual email body") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestHTMLToMarkdownFlattensEmailTables(t *testing.T) {
	markdown, err := HTMLToMarkdown(`<table><tr><td><h1>Title</h1></td></tr><tr><td><p>First cell</p></td><td><p>Second cell</p></td></tr></table>`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(markdown, "| |") {
		t.Fatalf("expected table layout to flatten, got %q", markdown)
	}
	for _, want := range []string{"Title", "First cell", "Second cell"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, missing %q", markdown, want)
		}
	}
}

func TestRenderEmptyBody(t *testing.T) {
	rendered, err := Render("", "", 80)
	if err != nil {
		t.Fatal(err)
	}
	if rendered != emptyBody {
		t.Fatalf("rendered = %q, want %q", rendered, emptyBody)
	}
}

func TestDecodeQuotedPrintable(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"hello", "hello"},
		{"=C2=A0", "\u00a0"},
		{"=E2=80=8C", "\u200c"},
		{"foo=\nbar", "foobar"},
		{"foo=\r\nbar", "foobar"},
		{"Price = $10", "Price = $10"},
		{"her=C2=A0tech.", "her\u00a0tech."},
	}
	for _, c := range cases {
		got := DecodeQuotedPrintable(c.input)
		if got != c.want {
			t.Errorf("DecodeQuotedPrintable(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
