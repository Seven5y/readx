package ui

import "github.com/charmbracelet/lipgloss"

// Warm, reading-friendly color palette.
var (
	Primary    = lipgloss.Color("#F5E6D3") // warm cream background
	Secondary  = lipgloss.Color("#8B7355") // muted brown
	Accent     = lipgloss.Color("#C8A96E") // gold accent
	MutedBg    = lipgloss.Color("#2C2416") // dark warm background
	Highlight  = lipgloss.Color("#E8D5B7") // light highlight
	DimText    = lipgloss.Color("#6B5D4F") // dimmed text
	BorderC    = lipgloss.Color("#5C4A3A") // border color
	PopupBg    = lipgloss.Color("#3D3226") // popup background
)

// HeaderStyle is for the top header bar.
var HeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Highlight).
	Background(MutedBg).
	Padding(0, 1).
	Width(80).Align(lipgloss.Center) // width is set dynamically

// HeaderBorder is the decorative line below the header.
var HeaderBorder = lipgloss.NewStyle().
	Foreground(BorderC).
	Background(MutedBg)

// FooterStyle is for the bottom status bar.
var FooterStyle = lipgloss.NewStyle().
	Foreground(DimText).
	Background(MutedBg).
	Padding(0, 1)

// ContentStyle is for the main text reading area.
var ContentStyle = lipgloss.NewStyle().
	Foreground(Primary).
	Background(MutedBg).
	Padding(0, 2)

// ContentPageIndicator shows the current page number within a chapter.
var ContentPageIndicator = lipgloss.NewStyle().
	Foreground(DimText).
	Background(MutedBg)

// PopupStyle is for the overlay dialog box.
var PopupStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Accent).
	Background(PopupBg).
	Foreground(Highlight).
	Padding(1, 2)

// PopupPrompt is for the Y/N text inside the popup.
var PopupPrompt = lipgloss.NewStyle().
	Foreground(Accent).
	Bold(true)

// ChapterModalStyle is the bordered container for the chapter list modal.
var ChapterModalStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Accent).
	Background(PopupBg).
	Foreground(Highlight).
	Padding(1, 2)

// ChapterModalTitleStyle is the title bar for the chapter list modal.
var ChapterModalTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Accent).
	Background(PopupBg).
	Align(lipgloss.Center)

// ChapterListItem styles unselected chapter entries in the modal.
var ChapterListItem = lipgloss.NewStyle().
	Foreground(Secondary).
	Background(PopupBg)

// ChapterListItemHighlight styles the cursor-highlighted chapter in the modal.
var ChapterListItemHighlight = lipgloss.NewStyle().
	Foreground(Accent).
	Background(PopupBg).
	Bold(true)

// LibraryTitleStyle is the title bar for the library shelf.
var LibraryTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Accent).
	Padding(0, 1)

// LibraryHintStyle is for the bottom hint bar in the library view.
var LibraryHintStyle = lipgloss.NewStyle().
	Foreground(DimText).
	Padding(0, 1)

// SearchInputStyle wraps the search keyword input inside the search modal.
var SearchInputStyle = lipgloss.NewStyle().
	Foreground(Primary).
	Background(PopupBg)

// SearchResultNormal styles unselected search result lines.
var SearchResultNormal = lipgloss.NewStyle().
	Foreground(Secondary).
	Background(PopupBg)

// SearchResultHighlight styles the cursor-highlighted search result.
var SearchResultHighlight = lipgloss.NewStyle().
	Foreground(Accent).
	Background(PopupBg).
	Bold(true)

// SearchGroupHeader styles chapter grouping headers in search results.
var SearchGroupHeader = lipgloss.NewStyle().
	Foreground(Accent).
	Background(PopupBg).
	Bold(true)

// SearchStatusMessage styles loading / no-results / truncated messages.
var SearchStatusMessage = lipgloss.NewStyle().
	Foreground(DimText).
	Background(PopupBg)
