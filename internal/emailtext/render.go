package emailtext

import (
	"fmt"
	"io"
	"mime/quotedprintable"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
)

const emptyBody = "(body not cached)"
const maxDisplayURLLength = 88

var whitespaceRE = regexp.MustCompile(`[ \t\r\f\v]+`)
var cssPropertyRE = regexp.MustCompile(`(?m)^\s*[-a-zA-Z]+\s*:\s*[^;]+;\s*$`)
var plainURLRE = regexp.MustCompile(`https?://[^\s<>()"']+`)

type Link struct {
	Text string
	URL  string
}

func DecodeQuotedPrintable(s string) string {
	if s == "" {
		return ""
	}
	// Soft line breaks are a strong signal of Quoted-Printable.
	if !strings.Contains(s, "=\n") && !strings.Contains(s, "=\r\n") {
		count := 0
		for i := 0; i < len(s)-2; i++ {
			if s[i] == '=' && isHex(s[i+1]) && isHex(s[i+2]) {
				count++
				if count >= 2 {
					break
				}
			}
		}
		if count < 2 {
			return s
		}
	}
	r := quotedprintable.NewReader(strings.NewReader(s))
	dec, err := io.ReadAll(r)
	if err != nil {
		return s
	}
	if !utf8.Valid(dec) {
		return s
	}
	return string(dec)
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'F') || (b >= 'a' && b <= 'f')
}

func Render(textBody, htmlBody string, width int) (string, error) {
	if width < 20 {
		width = 20
	}
	textBody = DecodeQuotedPrintable(textBody)
	htmlBody = DecodeQuotedPrintable(htmlBody)
	if MeaningfulPlainText(textBody) {
		return RenderMarkdown(normalizePlainText(textBody), width)
	}
	if strings.TrimSpace(htmlBody) == "" {
		return emptyBody, nil
	}
	markdown, err := HTMLToMarkdown(htmlBody)
	if err != nil {
		return "", err
	}
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return emptyBody, nil
	}
	return RenderMarkdown(markdown, width)
}

func MeaningfulPlainText(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if looksLikeCSSDump(value) {
		return false
	}
	withoutFiller := strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\u034f', '\u00ad':
			return -1
		}
		return r
	}, value)
	withoutFiller = strings.TrimSpace(strings.ReplaceAll(withoutFiller, "\u00a0", " "))
	return len([]rune(withoutFiller)) >= 3
}

func looksLikeCSSDump(value string) bool {
	trimmed := strings.TrimSpace(value)
	if looksLikeMinifiedCSS(trimmed) {
		return true
	}
	lines := strings.Split(value, "\n")
	nonEmpty := 0
	cssLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		nonEmpty++
		if strings.HasPrefix(line, "@media") || strings.HasPrefix(line, "@font-face") || strings.HasPrefix(line, ".") || strings.HasPrefix(line, "#") || strings.Contains(line, "{") || strings.Contains(line, "}") || cssPropertyRE.MatchString(line) {
			cssLines++
		}
		if nonEmpty >= 40 {
			break
		}
	}
	return nonEmpty >= 8 && cssLines*100/nonEmpty >= 60
}

func looksLikeMinifiedCSS(value string) bool {
	if len(value) < 500 || strings.Contains(value, "\n") {
		return false
	}
	if !strings.HasPrefix(value, ".") && !strings.HasPrefix(value, "#") && !strings.HasPrefix(value, "@") {
		return false
	}
	braces := strings.Count(value, "{") + strings.Count(value, "}")
	semicolons := strings.Count(value, ";")
	return braces >= 20 && semicolons >= 20
}

func HTMLToMarkdown(htmlBody string) (string, error) {
	cleaned, err := cleanHTML(htmlBody)
	if err != nil {
		return "", err
	}
	converter := md.NewConverter("", true, &md.Options{
		HeadingStyle:     "atx",
		BulletListMarker: "-",
		LinkStyle:        "inlined",
	})
	converter.Remove("script", "style", "noscript", "svg")
	converter.AddRules(
		md.Rule{Filter: []string{"a"}, Replacement: linkReplacement},
		md.Rule{Filter: []string{"table", "tbody", "thead", "tfoot", "tr"}, Replacement: blockReplacement},
		md.Rule{Filter: []string{"td", "th"}, Replacement: cellReplacement},
		md.Rule{Filter: []string{"img"}, Replacement: imageReplacement},
	)
	markdown, err := converter.ConvertString(cleaned)
	if err != nil {
		return "", err
	}
	return normalizeMarkdown(markdown), nil
}

func Links(textBody, htmlBody string) []Link {
	links := []Link{}
	seen := map[string]bool{}
	textBody = DecodeQuotedPrintable(textBody)
	htmlBody = DecodeQuotedPrintable(htmlBody)
	for _, link := range htmlLinks(htmlBody) {
		if link.URL == "" || seen[link.URL] {
			continue
		}
		seen[link.URL] = true
		links = append(links, link)
	}
	for _, rawURL := range plainURLRE.FindAllString(textBody, -1) {
		rawURL = strings.TrimRight(rawURL, ".,;:]")
		if rawURL == "" || seen[rawURL] {
			continue
		}
		seen[rawURL] = true
		links = append(links, Link{Text: displayURL(rawURL), URL: rawURL})
	}
	return links
}

func htmlLinks(htmlBody string) []Link {
	if strings.TrimSpace(htmlBody) == "" {
		return nil
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return nil
	}
	links := []Link{}
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		rawURL := strings.TrimSpace(attr(s, "href"))
		if rawURL == "" || strings.HasPrefix(rawURL, "#") || strings.HasPrefix(strings.ToLower(rawURL), "mailto:") {
			return
		}
		label := strings.TrimSpace(whitespaceRE.ReplaceAllString(s.Text(), " "))
		if label == "" {
			label = displayURL(rawURL)
		}
		links = append(links, Link{Text: label, URL: rawURL})
	})
	return links
}

func cleanHTML(htmlBody string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return "", err
	}
	doc.Find("script, style, noscript, svg").Remove()
	doc.Find("[hidden], [aria-hidden='true']").Remove()
	doc.Find(".preview, .preheader").Remove()
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		width := strings.TrimSpace(attr(s, "width"))
		height := strings.TrimSpace(attr(s, "height"))
		style := strings.ToLower(attr(s, "style"))
		if (width == "1" && height == "1") || strings.Contains(style, "display:none") || strings.Contains(style, "display: none") {
			s.Remove()
		}
	})
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		style := strings.ToLower(attr(s, "style"))
		if strings.Contains(style, "display:none") || strings.Contains(style, "display: none") || strings.Contains(style, "visibility:hidden") || strings.Contains(style, "visibility: hidden") || strings.Contains(style, "max-height:0") || strings.Contains(style, "max-height: 0") {
			s.Remove()
		}
	})
	body := doc.Find("body")
	if body.Length() > 0 {
		return body.Html()
	}
	return doc.Html()
}

func linkReplacement(content string, selec *goquery.Selection, _ *md.Options) *string {
	label := strings.TrimSpace(stripMarkdownNoise(content))
	href := strings.TrimSpace(attr(selec, "href"))
	if href == "" {
		return md.String(label)
	}
	href = displayURL(href)
	if label == "" || label == href {
		return md.String(href)
	}
	return md.String(fmt.Sprintf("%s (%s)", label, href))
}

func displayURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return truncateString(rawURL, maxDisplayURLLength)
	}
	query := parsed.Query()
	for key := range query {
		if strings.HasPrefix(strings.ToLower(key), "utm_") || isTrackingQueryKey(key) {
			query.Del(key)
		}
	}
	parsed.RawQuery = query.Encode()
	cleaned := parsed.String()
	if len(cleaned) <= maxDisplayURLLength {
		return cleaned
	}
	parsed.RawQuery = ""
	cleaned = parsed.String()
	return truncateString(cleaned, maxDisplayURLLength)
}

func isTrackingQueryKey(key string) bool {
	switch strings.ToLower(key) {
	case "token", "email", "inbox", "utm", "r", "s", "source", "medium", "campaign":
		return true
	default:
		return false
	}
}

func blockReplacement(content string, _ *goquery.Selection, _ *md.Options) *string {
	content = strings.TrimSpace(content)
	if content == "" {
		return md.String("")
	}
	return md.String("\n\n" + content + "\n\n")
}

func cellReplacement(content string, _ *goquery.Selection, _ *md.Options) *string {
	content = strings.TrimSpace(content)
	if content == "" {
		return md.String("")
	}
	return md.String(content + "\n")
}

func imageReplacement(_ string, selec *goquery.Selection, _ *md.Options) *string {
	alt := strings.TrimSpace(attr(selec, "alt"))
	if alt == "" {
		return md.String("")
	}
	return md.String(alt)
}

func RenderMarkdown(markdown string, width int) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return "", err
	}
	rendered, err := renderer.Render(markdown)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(rendered), nil
}

func attr(selec *goquery.Selection, name string) string {
	value, _ := selec.Attr(name)
	return value
}

func normalizePlainText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
}

func normalizeMarkdown(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	compact := make([]string, 0, len(lines))
	blanks := 0
	for _, line := range lines {
		line = strings.TrimSpace(whitespaceRE.ReplaceAllString(line, " "))
		line = stripMarkdownNoise(line)
		if line == "" {
			blanks++
			if blanks > 1 {
				continue
			}
		} else {
			blanks = 0
		}
		compact = append(compact, line)
	}
	return strings.TrimSpace(strings.Join(compact, "\n"))
}

func stripMarkdownNoise(value string) string {
	value = strings.ReplaceAll(value, "| |", " ")
	value = strings.Trim(value, "| ")
	value = strings.TrimSpace(value)
	return value
}

func truncateString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}
