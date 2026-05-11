package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"readx/internal/domain"
)

// BodyView renders the main body area: full-width content panel.
func BodyView(page domain.Page, termWidth, termHeight int, bgColor lipgloss.Color, searchQuery string) string {
	headerH := 3
	footerH := 1
	bodyHeight := termHeight - headerH - footerH
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	return buildContent(page, termWidth, bodyHeight, bgColor, searchQuery)
}

// buildContent renders the current page text.
func buildContent(page domain.Page, width, height int, bgColor lipgloss.Color, searchQuery string) string {
	var contentLines []string

	baseContent := ContentStyle.Width(width).Background(bgColor)
	styledContent := baseContent
	accentContent := baseContent.Foreground(Accent)
	lowerQuery := strings.ToLower(searchQuery)

	for _, line := range page.Lines {
		if searchQuery != "" && strings.Contains(strings.ToLower(line), lowerQuery) {
			contentLines = append(contentLines, accentContent.Render(line))
		} else {
			contentLines = append(contentLines, styledContent.Render(line))
		}
	}

	for len(contentLines) < height {
		contentLines = append(contentLines, styledContent.Render(""))
	}

	if len(contentLines) > height {
		contentLines = contentLines[:height]
	}

	if page.TotalInChapter > 1 {
		indicator := fmt.Sprintf("— pg %d/%d —", page.PageIndex+1, page.TotalInChapter)
		indicatorLine := ContentPageIndicator.Width(width).Background(bgColor).Align(lipgloss.Right).Render(indicator)
		if strings.TrimSpace(contentLines[height-1]) == "" {
			contentLines[height-1] = indicatorLine
		}
	}

	return lipgloss.JoinVertical(lipgloss.Top, contentLines...)
}
