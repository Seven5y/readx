package ui

import "github.com/mattn/go-runewidth"

// visibleWindow returns the start and end indices for a centered scrolling
// viewport of size viewportSize into a list of totalItems, centered on cursor.
func visibleWindow(totalItems, cursor, viewportSize int) (start, end int) {
	half := viewportSize / 2
	start = cursor - half
	if start < 0 {
		start = 0
	}
	end = start + viewportSize
	if end > totalItems {
		end = totalItems
		start = max(0, end-viewportSize)
	}
	return start, end
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
