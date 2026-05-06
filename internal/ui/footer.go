package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FooterView renders the bottom status bar with three columns:
// left = mode indicator or command input, center = progress bar,
// right = page number and hints.
func FooterView(curChapter, totalChapters, curPage, totalPages, termWidth int, commandMode bool, cmdInputView string) string {
	if termWidth < 20 {
		termWidth = 20
	}

	progress := calcProgress(curChapter, totalChapters, curPage, totalPages)
	bar := progressBar(progress, 10)

	if commandMode {
		// Command mode: left = input, center = hidden, right = hints.
		rightText := fmt.Sprintf("第%d/%d页  enter执行 esc取消", curPage+1, totalPages)
		rightW := lipgloss.Width(rightText)
		inputW := termWidth - rightW
		if inputW < 5 {
			inputW = 5
		}
		leftSide := FooterStyle.Width(inputW).Render(cmdInputView)
		rightSide := FooterStyle.Align(lipgloss.Right).Render(rightText)
		return lipgloss.JoinHorizontal(lipgloss.Top, leftSide, rightSide)
	}

	// Normal mode: left = indicator + bar, right = page + hints.
	leftText := "[阅读]  " + bar
	rightText := fmt.Sprintf("第%d/%d页  tab目录 q退出", curPage+1, totalPages)
	rightW := lipgloss.Width(rightText)
	leftSide := FooterStyle.Width(termWidth - rightW).Render(leftText)
	rightSide := FooterStyle.Align(lipgloss.Right).Render(rightText)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftSide, rightSide)
}

// calcProgress computes an approximate reading progress percentage.
func calcProgress(curCh, totalCh, curPage, totalPages int) int {
	if totalCh == 0 {
		return 0
	}

	chapterWeight := 100.0 / float64(totalCh)
	progress := float64(curCh) * chapterWeight
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
