package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HeaderView renders the top header bar with book title and current chapter.
// It occupies 3 lines: top border, title line, chapter line.
func HeaderView(bookTitle, chapterTitle string, width int) string {
	if width < 20 {
		width = 20
	}

	border := HeaderBorder.Render(strings.Repeat("─", width))
	titleLine := HeaderStyle.Width(width).Align(lipgloss.Center).Render(bookTitle)
	chapterLine := HeaderStyle.Width(width).Align(lipgloss.Center).Foreground(Secondary).
		Render(fmt.Sprintf("— %s —", chapterTitle))

	return lipgloss.JoinVertical(lipgloss.Top, border, titleLine, chapterLine)
}
