package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"readx/internal/domain"
	"readx/internal/persistence"
)

// LibraryItem implements list.Item for the bookshelf.
type LibraryItem struct {
	Entry domain.LibraryEntry
}

func (i LibraryItem) Title() string       { return i.Entry.Title }
func (i LibraryItem) Description() string { return i.desc() }
func (i LibraryItem) FilterValue() string { return i.Entry.Title }

func (i LibraryItem) desc() string {
	return fmt.Sprintf("%d%%  %s", i.Entry.Progress, timeAgo(i.Entry.LastRead))
}

// LibraryModel is the bookshelf view using bubbles/list.
type LibraryModel struct {
	list   list.Model
	config *persistence.Config
}

// NewLibraryModel creates a library model with the given config.
func NewLibraryModel(config *persistence.Config) *LibraryModel {
	entries := persistence.ListBooks(config)
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = LibraryItem{Entry: e}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(Accent).
		BorderForeground(Accent)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(Primary)

	l := list.New(items, delegate, 0, 0)
	l.Title = "书架"
	l.Styles.Title = LibraryTitleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return &LibraryModel{list: l, config: config}
}

func (m *LibraryModel) Init() tea.Cmd { return nil }

func (m *LibraryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-2)

	case tea.KeyMsg:
		switch msg.String() {

		case "enter":
			if i, ok := m.list.SelectedItem().(LibraryItem); ok {
				return m, func() tea.Msg { return OpenBookMsg{Path: i.Entry.Path} }
			}

		case "d":
			if i, ok := m.list.SelectedItem().(LibraryItem); ok {
				if err := persistence.RemoveBook(m.config, i.Entry.Path); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: remove book: %v\n", err)
				}
				// Refresh list.
				entries := persistence.ListBooks(m.config)
				items := make([]list.Item, len(entries))
				for j, e := range entries {
					items[j] = LibraryItem{Entry: e}
				}
				m.list.SetItems(items)
			}

		case "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *LibraryModel) View() string {
	return m.list.View() + "\n" +
		LibraryHintStyle.Width(m.list.Width()).Render("Enter 阅读  d 移除  q 退出")
}

// timeAgo returns a Chinese relative time string for the given time.
func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "未阅读"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d小时前", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d天前", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%d周前", int(d.Hours()/(24*7)))
	default:
		return "很久以前"
	}
}
