package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FooterView renders the bottom status bar.
// Left side: key help text. Right side: character-style progress bar.
func FooterView(currentChapter, totalChapters, currentPage, totalPages int, width int) string {
	if width < 20 {
		width = 20
	}

	help := "↑/↓翻页  ←/→章节  tab目录  q退出"

	progress := calcProgress(currentChapter, totalChapters, currentPage, totalPages)
	rightText := progressBar(progress, 12)

	// Left-align help, right-align progress bar.
	rightW := lipgloss.Width(rightText)
	leftSide := FooterStyle.Width(width - rightW).Render(help)
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

// progressBar renders a character-style progress bar: [████░░░░░░] 40%.
func progressBar(pct int, length int) string {
	if length < 2 {
		length = 2
	}
	filled := pct * length / 100
	if filled > length {
		filled = length
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", length-filled)
	return fmt.Sprintf("[%s] %d%%", bar, pct)
}
