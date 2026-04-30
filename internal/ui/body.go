package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"readx/internal/domain"
)

// BodyView renders the main body area: left sidebar (chapter list) and right content panel.
// When showSidebar is false, the content panel expands to full width.
func BodyView(chapterTitles []string, curChapter int, page domain.Page, showSidebar bool, termWidth, termHeight int) string {
	headerH := 3
	footerH := 1
	bodyHeight := termHeight - headerH - footerH
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	if showSidebar {
		sidebarW := max(1, int(float64(termWidth)*0.2))
		contentW := termWidth - sidebarW

		// Build sidebar with scrollable chapter list.
		sidebar := buildSidebar(chapterTitles, curChapter, sidebarW, bodyHeight)

		// Build content panel.
		content := buildContent(page, contentW, bodyHeight)

		return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	}

	// Full-width content.
	return buildContent(page, termWidth, bodyHeight)
}

// buildSidebar renders the chapter list in the left panel.
func buildSidebar(titles []string, curChapter, width, height int) string {
	// Calculate visible range: center the current chapter when possible.
	half := height / 2
	start := curChapter - half
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(titles) {
		end = len(titles)
		start = max(0, end-height)
	}

	var lines []string
	for i := start; i < end; i++ {
		line := truncateToWidth(titles[i], width)
		if i == curChapter {
			line = SidebarHighlight.Width(width).Render(line)
		} else {
			line = SidebarStyle.Width(width).Render(line)
		}
		lines = append(lines, line)
	}

	// Pad to fill the full height.
	for len(lines) < height {
		lines = append(lines, SidebarStyle.Width(width).Render(""))
	}

	return lipgloss.JoinVertical(lipgloss.Top, lines...)
}

// buildContent renders the current page text in the right panel.
func buildContent(page domain.Page, width, height int) string {
	var contentLines []string

	// Render page lines. Pre-compute the width-styled style once.
	styledContent := ContentStyle.Width(width)
	for _, line := range page.Lines {
		contentLines = append(contentLines, styledContent.Render(line))
	}

	// Fill remaining space.
	for len(contentLines) < height {
		contentLines = append(contentLines, styledContent.Render(""))
	}

	// If there are more lines than available height, truncate.
	if len(contentLines) > height {
		contentLines = contentLines[:height]
	}

	// Add a page indicator at the bottom-right when there are multiple pages.
	if page.TotalInChapter > 1 {
		indicator := fmt.Sprintf("— pg %d/%d —", page.PageIndex+1, page.TotalInChapter)
		indicatorLine := ContentPageIndicator.Width(width).Align(lipgloss.Right).Render(indicator)
		// Replace the last line if it's empty, otherwise append.
		if strings.TrimSpace(contentLines[height-1]) == "" {
			contentLines[height-1] = indicatorLine
		}
	}

	return lipgloss.JoinVertical(lipgloss.Top, contentLines...)
}

// truncateToWidth truncates a string to fit the given display width,
// correctly accounting for CJK characters that occupy 2 columns.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	col := 0
	for i, r := range runes {
		rw := runewidth.RuneWidth(r)
		if col+rw > width {
			if width > 3 {
				return string(runes[:i]) + "…"
			}
			return string(runes[:i])
		}
		col += rw
	}
	return s
}
