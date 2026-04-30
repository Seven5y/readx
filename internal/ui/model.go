// Package ui implements the Bubble Tea TUI for the readx terminal reader.
package ui

import (
	"github.com/charmbracelet/bubbletea"

	"fmt"
	"os"

	"readx/internal/domain"
	"readx/internal/persistence"
	"readx/internal/service"
)

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

// paginateChapterCmd is a bubbletea.Cmd that paginates a chapter and returns
// the result wrapped in a paginateDoneMsg.
func paginateChapterCmd(reader domain.Reader, cache *service.PageCache, chapterIndex, termW, termH int) tea.Cmd {
	return func() tea.Msg {
		pages, err := service.PaginateOrCache(cache, reader, chapterIndex, termW, termH)
		if err != nil {
			return paginateErrMsg{err}
		}
		return paginateDoneMsg{
			chapterIndex: chapterIndex,
			pages:        pages,
		}
	}
}

type paginateDoneMsg struct {
	chapterIndex int
	pages        []domain.Page
}

type paginateErrMsg struct {
	err error
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model is the top-level Bubble Tea model for the reader UI.
type Model struct {
	reader        domain.Reader
	cache         *service.PageCache
	chapterTitles []string

	curChapter int
	curPage    int // page index within current chapter
	numPages   int // pages in current chapter

	showPopup   bool
	popupMsg    string

	showChapters  bool // chapter list modal visible
	chapterCursor int  // highlighted chapter index in modal

	ready bool

	termWidth  int
	termHeight int

	// Persisted progress for save-on-quit.
	config   *persistence.Config
	bookPath string
}

// NewModel creates a new reader UI model.
// If savedProgress is non-nil, a "continue reading?" popup will be shown.
func NewModel(reader domain.Reader, config *persistence.Config, chapterTitles []string, savedProgress *domain.ReadingProgress) *Model {
	m := &Model{
		reader:        reader,
		cache:         service.NewPageCache(),
		chapterTitles: chapterTitles,
		curChapter:    0,
		curPage:       0,
		config:        config,
		bookPath:      reader.GetBook().Path,
	}

	if savedProgress != nil {
		m.curChapter = savedProgress.ChapterIndex
		m.curPage = savedProgress.PageIndex
		m.showPopup = true
		m.popupMsg = "检测到历史进度，\n是否继续阅读？\n\n[Y] 是  [N] 否"
	}

	return m
}

// ---------------------------------------------------------------------------
// tea.Model interface
// ---------------------------------------------------------------------------

// Init returns the initial command. Actual pagination is deferred until
// the first WindowSizeMsg arrives so we have real terminal dimensions.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages (key presses, window resize, async results).
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		if m.ready {
			return m, m.repaginate()
		}
		m.ready = true
		return m, m.repaginate()

	case paginateDoneMsg:
		m.ready = true
		if msg.chapterIndex == m.curChapter {
			m.numPages = len(msg.pages)
			// Clamp page index after repagination.
			if m.curPage >= m.numPages {
				m.curPage = max(0, m.numPages-1)
			}
		}
		return m, nil

	case paginateErrMsg:
		m.ready = true
		fmt.Fprintf(os.Stderr, "Pagination error: %v\n", msg.err)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// View renders the full TUI.
func (m *Model) View() string {
	// Show a "parsing…" placeholder until the first chapter is paginated.
	if !m.ready {
		return "正在解析…\n"
	}

	// Popup overlay takes priority.
	if m.showPopup {
		return PopupView(m.popupMsg, m.termWidth, m.termHeight)
	}

	// Chapter list modal replaces the screen.
	if m.showChapters {
		return ChapterListView(m.chapterTitles, m.chapterCursor, m.termWidth, m.termHeight)
	}

	// Build the main layout.
	book := m.reader.GetBook()
	chapterTitle := ""
	if m.curChapter < len(m.chapterTitles) {
		chapterTitle = m.chapterTitles[m.curChapter]
	}

	header := HeaderView(book.Title, chapterTitle, m.termWidth)

	// Get current page data.
	page := domain.Page{
		Lines:          []string{"(loading…)"},
		ChapterIndex:   m.curChapter,
		PageIndex:      m.curPage,
		TotalInChapter: m.numPages,
	}
	if cached := m.cache.Get(m.curChapter); cached != nil {
		if m.curPage < len(cached) {
			page = cached[m.curPage]
		} else if len(cached) > 0 {
			page = cached[0]
		}
	}

	body := BodyView(page, m.termWidth, m.termHeight)
	footer := FooterView(m.curChapter, m.reader.GetTotalChapters(), m.curPage, m.numPages, m.termWidth)

	return header + "\n" + body + "\n" + footer
}

// Cleanup should be called after the Bubble Tea program exits to persist state.
func (m *Model) Cleanup() {
	if m.config == nil {
		return
	}
	if err := persistence.SaveProgress(m.config, m.bookPath, domain.ReadingProgress{
		BookPath:     m.bookPath,
		ChapterIndex: m.curChapter,
		PageIndex:    m.curPage,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: save progress: %v\n", err)
	}
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

// repaginate triggers async pagination for the current chapter.
func (m *Model) repaginate() tea.Cmd {
	return paginateChapterCmd(m.reader, m.cache, m.curChapter, m.termWidth, m.termHeight)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Tier 1: Popup mode — only respond to Y/N.
	if m.showPopup {
		switch msg.String() {
		case "y", "Y":
			m.showPopup = false
			return m, m.repaginate()
		case "n", "N":
			m.showPopup = false
			m.curChapter = 0
			m.curPage = 0
			return m, m.repaginate()
		}
		return m, nil
	}

	// Tier 2: Chapter list modal mode.
	if m.showChapters {
		switch msg.String() {
		case "up", "k":
			if m.chapterCursor > 0 {
				m.chapterCursor--
			}
		case "down", "j":
			if m.chapterCursor < len(m.chapterTitles)-1 {
				m.chapterCursor++
			}
		case "enter":
			return m.gotoChapter(m.chapterCursor)
		case "esc", "tab":
			m.showChapters = false
		}
		return m, nil
	}

	// Tier 3: Normal reading mode.
	switch msg.String() {

	case "q":
		m.Cleanup()
		return m, tea.Quit

	case "up", "k":
		return m.prevPage()

	case "down", "j":
		return m.nextPage()

	case "left", "h":
		return m.prevChapter()

	case "right", "l":
		return m.nextChapter()

	case "tab":
		m.showChapters = true
		m.chapterCursor = m.curChapter
	}

	return m, nil
}

// gotoChapter jumps to the target chapter, resets page to 0, closes the
// chapter modal, and triggers pagination for the new chapter.
func (m *Model) gotoChapter(targetChapter int) (tea.Model, tea.Cmd) {
	if targetChapter == m.curChapter {
		m.showChapters = false
		return m, nil
	}
	m.curChapter = targetChapter
	m.curPage = 0
	m.showChapters = false
	return m, m.repaginate()
}

// ---------------------------------------------------------------------------
// Navigation helpers
// ---------------------------------------------------------------------------

func (m *Model) prevPage() (tea.Model, tea.Cmd) {
	if m.curPage > 0 {
		m.curPage--
		return m, nil
	}

	// At first page of current chapter — jump to prev chapter's last page.
	if m.curChapter > 0 {
		m.curChapter--
		return m, m.repaginate()
	}

	return m, nil
}

func (m *Model) nextPage() (tea.Model, tea.Cmd) {
	if m.curPage < m.numPages-1 {
		m.curPage++
		return m, nil
	}

	// At last page — jump to next chapter's first page.
	if m.curChapter < m.reader.GetTotalChapters()-1 {
		m.curChapter++
		m.curPage = 0
		return m, m.repaginate()
	}

	return m, nil
}

func (m *Model) prevChapter() (tea.Model, tea.Cmd) {
	if m.curChapter > 0 {
		m.curChapter--
		m.curPage = 0
		return m, m.repaginate()
	}
	return m, nil
}

func (m *Model) nextChapter() (tea.Model, tea.Cmd) {
	if m.curChapter < m.reader.GetTotalChapters()-1 {
		m.curChapter++
		m.curPage = 0
		return m, m.repaginate()
	}
	return m, nil
}
