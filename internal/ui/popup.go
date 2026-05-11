package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"readx/internal/domain"
	"readx/internal/persistence"
)

// PopupView renders a centered overlay dialog. Used for the
// "Continue reading?" prompt when a saved progress is detected.
func PopupView(message string, width, height int, bgColor lipgloss.Color) string {
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}

	// Build the popup content.
	styledPrompt := PopupPrompt.Background(bgColor)
	msgLines := strings.Split(message, "\n")
	var contentLines []string
	for _, line := range msgLines {
		contentLines = append(contentLines, styledPrompt.Render(line))
	}
	content := lipgloss.JoinVertical(lipgloss.Center, contentLines...)

	popupW := min(width-8, 60)
	popupH := len(msgLines) + 4

	// Render the bordered popup.
	popup := PopupStyle.Width(popupW).Height(popupH).Background(bgColor).Align(lipgloss.Center).Render(content)

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
func ChapterListView(titles []string, cursor, width, height int, bgColor lipgloss.Color) string {
	items := make([]string, len(titles))
	for i, t := range titles {
		items[i] = truncateToWidth(t, 36)
	}
	return renderListModal("[ 章节目录 ]", items, cursor, width, height, bgColor)
}

// ---------------------------------------------------------------------------
// Settings panel
// ---------------------------------------------------------------------------

// settingItem represents a single configurable setting.
type settingItem struct {
	label string
	get   func(*persistence.UserSettings) string
	set   func(*persistence.UserSettings)
}

// settingsItems is the registry of all configurable settings.
var settingsItems = []settingItem{
	{
		label: "背景色",
		get:   func(s *persistence.UserSettings) string { return boolLabel(s.BgColor) },
		set:   func(s *persistence.UserSettings) { s.BgColor = !s.BgColor },
	},
}

func boolLabel(v bool) string {
	if v {
		return "开启"
	}
	return "关闭"
}

// ConfigPanelView renders a centered settings panel for /config command.
func ConfigPanelView(settings *persistence.UserSettings, cursor, width, height int, bgColor lipgloss.Color) string {
	items := make([]string, len(settingsItems))
	for i, si := range settingsItems {
		items[i] = truncateToWidth(si.label+": "+si.get(settings), 36)
	}
	return renderListModal("[ 设置 ]", items, cursor, width, height, bgColor)
}

// ---------------------------------------------------------------------------
// Shared modal rendering
// ---------------------------------------------------------------------------

// renderListModal renders a centered, scrollable list modal with a title.
// Callers pre-format their items as []string.
func renderListModal(title string, items []string, cursor, width, height int, bgColor lipgloss.Color) string {
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

	normalStyle := ChapterListItem.Background(bgColor)
	highlightStyle := ChapterListItemHighlight.Background(PopupBg)

	start, end := visibleWindow(len(items), cursor, viewportH)

	var lines []string
	for i := start; i < end; i++ {
		if i == cursor {
			lines = append(lines, highlightStyle.Render(" "+items[i]+" "))
		} else {
			lines = append(lines, normalStyle.Render(" "+items[i]+" "))
		}
	}

	for len(lines) < viewportH {
		lines = append(lines, normalStyle.Render(""))
	}

	titleLine := ChapterModalTitleStyle.Background(bgColor).Render(title)
	inner := lipgloss.JoinVertical(lipgloss.Top, titleLine, "", lipgloss.JoinVertical(lipgloss.Top, lines...))

	modal := ChapterModalStyle.Background(bgColor).Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, modal)
}

// ---------------------------------------------------------------------------
// Search modal
// ---------------------------------------------------------------------------

type searchDisplayItem struct {
	text        string
	isHeader    bool
	resultIndex int // index into searchResults; -1 for headers
}

func searchDisplayItems(results []domain.SearchResult) []searchDisplayItem {
	var items []searchDisplayItem
	lastChapter := -1
	for i, r := range results {
		if r.ChapterIndex != lastChapter {
			items = append(items, searchDisplayItem{
				text:        "── " + r.ChapterTitle + " ──",
				isHeader:    true,
				resultIndex: -1,
			})
			lastChapter = r.ChapterIndex
		}
		items = append(items, searchDisplayItem{
			text:        truncateToWidth(r.LineContent, 40),
			isHeader:    false,
			resultIndex: i,
		})
	}
	return items
}

func findCursorDisplayIdx(items []searchDisplayItem, cursor int) int {
	for i, item := range items {
		if !item.isHeader && item.resultIndex == cursor {
			return i
		}
	}
	return 0
}

func SearchView(inputView string, results []domain.SearchResult, cursor int, loading, truncated bool, termWidth, termHeight int, bgColor lipgloss.Color) string {
	if termWidth < 40 {
		termWidth = 40
	}
	if termHeight < 14 {
		termHeight = 14
	}

	titleLine := ChapterModalTitleStyle.Background(bgColor).Render("[ 搜索 ]")
	styledInput := SearchInputStyle.Background(bgColor).Width(50).Render(inputView)

	viewportH := termHeight - 12
	if viewportH < 3 {
		viewportH = 3
	}

	var resultLines []string

	if loading {
		resultLines = append(resultLines, SearchStatusMessage.Render("搜索中…"))
	} else if len(results) == 0 {
		resultLines = append(resultLines, SearchStatusMessage.Render("未找到匹配结果"))
	} else {
		items := searchDisplayItems(results)
		displayIdx := findCursorDisplayIdx(items, cursor)
		start, end := visibleWindow(len(items), displayIdx, viewportH)

		normalStyle := SearchResultNormal.Background(bgColor)
		highlightStyle := SearchResultHighlight.Background(PopupBg)

		for i := start; i < end && len(resultLines) < viewportH; i++ {
			item := items[i]
			if item.isHeader {
				resultLines = append(resultLines, SearchGroupHeader.Render(item.text))
			} else if i == displayIdx {
				resultLines = append(resultLines, highlightStyle.Render("  "+item.text+" "))
			} else {
				resultLines = append(resultLines, normalStyle.Render("  "+item.text+" "))
			}
		}
	}

	for len(resultLines) < viewportH {
		resultLines = append(resultLines, "")
	}

	if truncated {
		resultLines = append(resultLines, SearchStatusMessage.Render("…结果过多，仅显示前 200 条"))
	}

	inner := lipgloss.JoinVertical(lipgloss.Top, titleLine, "", styledInput, "", lipgloss.JoinVertical(lipgloss.Top, resultLines...))
	modal := ChapterModalStyle.Background(bgColor).Render(inner)

	return lipgloss.Place(termWidth, termHeight, lipgloss.Center, lipgloss.Center, modal)
}
