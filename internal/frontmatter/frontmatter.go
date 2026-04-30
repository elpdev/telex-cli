package frontmatter

import (
	"fmt"
	"sort"
	"strings"
)

type Document struct {
	Fields map[string]string
	Body   string
}

func Parse(content string) (Document, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	if !strings.HasPrefix(content, "---\n") && strings.TrimSpace(content) != "---" {
		return Document{Fields: map[string]string{}, Body: content}, nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return Document{Fields: map[string]string{}, Body: content}, nil
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return Document{}, fmt.Errorf("frontmatter is missing closing ---")
	}
	fields := map[string]string{}
	for lineNo, line := range lines[1:end] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Document{}, fmt.Errorf("invalid frontmatter line %d", lineNo+2)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return Document{}, fmt.Errorf("invalid frontmatter line %d", lineNo+2)
		}
		fields[key] = unquoteValue(strings.TrimSpace(value))
	}
	bodyStart := end + 1
	if bodyStart < len(lines) && strings.TrimSpace(lines[bodyStart]) == "" {
		bodyStart++
	}
	return Document{Fields: fields, Body: strings.Join(lines[bodyStart:], "\n")}, nil
}

func Render(fields map[string]string, body string) string {
	return RenderWithOrder(fields, nil, body)
}

func RenderWithOrder(fields map[string]string, order []string, body string) string {
	var b strings.Builder
	b.WriteString("---\n")
	keys := orderedKeys(fields, order)
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(quoteValue(fields[key]))
		b.WriteString("\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(body)
	return b.String()
}

func orderedKeys(fields map[string]string, order []string) []string {
	keys := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, key := range keys {
		seen[key] = true
	}
	for _, key := range order {
		if _, ok := fields[key]; ok && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	rest := make([]string, 0, len(fields)-len(keys))
	for key := range fields {
		if !seen[key] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func unquoteValue(value string) string {
	if len(value) >= 2 {
		quote := value[0]
		if (quote == '\'' || quote == '"') && value[len(value)-1] == quote {
			value = value[1 : len(value)-1]
		}
	}
	if strings.HasPrefix(value, "#") {
		return ""
	}
	return strings.TrimSpace(value)
}

func quoteValue(value string) string {
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, ":#\n\t\"'") || strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ") {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return value
}
