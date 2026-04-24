package modal

import (
	"github.com/elpdev/telex-cli/internal/theme"
	"charm.land/lipgloss/v2"
)

func Overlay(base, content string, width, height int, _ theme.Theme) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content, lipgloss.WithWhitespaceChars(" "))
}
