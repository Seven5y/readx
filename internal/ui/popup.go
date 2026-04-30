package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PopupView renders a centered overlay dialog. Used for the
// "Continue reading?" prompt when a saved progress is detected.
func PopupView(message string, width, height int) string {
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}

	// Build the popup content.
	msgLines := strings.Split(message, "\n")
	var contentLines []string
	for _, line := range msgLines {
		contentLines = append(contentLines, PopupPrompt.Render(line))
	}
	content := lipgloss.JoinVertical(lipgloss.Center, contentLines...)

	popupW := min(width-8, 60)
	popupH := len(msgLines) + 4 // add padding for prompt text

	// Render the bordered popup.
	popup := PopupStyle.Width(popupW).Height(popupH).Align(lipgloss.Center).Render(content)

	// Center the popup vertically and horizontally in the terminal.
	// lipgloss.Place handles padding to center the box.
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		popup,
	)
}

// ChapterListView renders a centered modal dialog listing all chapters
// for navigation. The current cursor position is highlighted.
func ChapterListView(titles []string, cursor, width, height int) string {
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}

	viewportH := height - 8
	if viewportH < 3 {
		viewportH = 3
	}

	start, end := visibleWindow(len(titles), cursor, viewportH)

	var items []string
	if len(titles) == 0 {
		items = append(items, ChapterListItem.Render(" (no chapters)"))
	} else {
		for i := start; i < end; i++ {
			line := truncateToWidth(titles[i], 36)
			if i == cursor {
				items = append(items, ChapterListItemHighlight.Render(" "+line+" "))
			} else {
				items = append(items, ChapterListItem.Render(" "+line+" "))
			}
		}
	}

	for len(items) < viewportH {
		items = append(items, ChapterListItem.Render(""))
	}

	titleLine := ChapterModalTitleStyle.Render("[ 章节目录 ]")
	inner := lipgloss.JoinVertical(lipgloss.Top, titleLine, "", lipgloss.JoinVertical(lipgloss.Top, items...))

	modal := ChapterModalStyle.Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}
