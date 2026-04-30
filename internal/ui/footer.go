package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// FooterView renders the bottom status bar.
// Left side: key help text. Right side: reading progress percentage.
func FooterView(currentChapter, totalChapters, currentPage, totalPages int, width int) string {
	if width < 20 {
		width = 20
	}

	// Build help text with highlighted keys.
	help := "↑/↓ page  ←/→ chapter  tab sidebar  q quit"

	// Calculate overall progress percentage.
	progress := calcProgress(currentChapter, totalChapters, currentPage, totalPages)
	rightText := fmt.Sprintf("%d%%", progress)

	// Left-align help, right-align progress.
	leftSide := FooterStyle.Width(width - len(rightText) - 2).Render(help)
	rightSide := FooterStyle.Align(lipgloss.Right).Render(rightText)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftSide, rightSide)
}

// calcProgress computes an approximate reading progress percentage.
func calcProgress(curCh, totalCh, curPage, totalPages int) int {
	if totalCh == 0 {
		return 0
	}

	// Weight: each chapter contributes equally.
	chapterWeight := 100.0 / float64(totalCh)
	progress := float64(curCh) * chapterWeight

	// Add partial progress within the current chapter.
	if totalPages > 0 {
		progress += chapterWeight * (float64(curPage) / float64(totalPages))
	}

	p := int(progress)
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return p
}
