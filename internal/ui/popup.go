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
