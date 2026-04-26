package tasks

import "strings"

func AddCardToColumn(markdown, column, cardPath string) string {
	column = strings.TrimSpace(column)
	cardPath = strings.TrimSpace(cardPath)
	if column == "" || cardPath == "" || strings.Contains(markdown, "[["+cardPath+"]]") {
		return markdown
	}
	line := "- [[" + cardPath + "]]"
	if strings.TrimSpace(markdown) == "" {
		return "# Board\n\n## " + column + "\n" + line + "\n"
	}
	lines := strings.Split(strings.TrimRight(markdown, "\n"), "\n")
	insertAt := -1
	found := false
	for i, value := range lines {
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(trimmed, "## ") {
			if found {
				insertAt = i
				break
			}
			if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), column) {
				found = true
				insertAt = i + 1
			}
		}
	}
	if !found {
		lines = append(lines, "", "## "+column, line)
		return strings.Join(lines, "\n") + "\n"
	}
	updated := make([]string, 0, len(lines)+1)
	updated = append(updated, lines[:insertAt]...)
	updated = append(updated, line)
	updated = append(updated, lines[insertAt:]...)
	return strings.Join(updated, "\n") + "\n"
}

func ReplaceCardPath(markdown, oldPath, newPath string) string {
	oldPath = strings.TrimSpace(oldPath)
	newPath = strings.TrimSpace(newPath)
	if oldPath == "" || newPath == "" || oldPath == newPath {
		return markdown
	}
	return strings.ReplaceAll(markdown, "[["+oldPath+"]]", "[["+newPath+"]]")
}
