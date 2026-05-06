// Package ui implements the Bubble Tea TUI for the readx terminal reader.
package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"fmt"
	"os"
	"strconv"
	"strings"

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

// ReaderModel is the Bubble Tea model for the reader UI.
type ReaderModel struct {
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

	commandMode bool            // true = command input active
	cmdInput    textinput.Model // command input box

	showConfig   bool // settings panel visible
	configCursor int  // cursor in settings panel
	configDirty  bool // true if settings were modified in this session

	ready bool

	termWidth  int
	termHeight int

	// Persisted progress for save-on-quit.
	config   *persistence.Config
	bookPath string
}

// NewReaderModel creates a new reader UI model.
// If savedProgress is non-nil, a "continue reading?" popup will be shown.
func NewReaderModel(reader domain.Reader, config *persistence.Config, chapterTitles []string, savedProgress *domain.ReadingProgress) *ReaderModel {
	cmdInput := textinput.New()
	cmdInput.Placeholder = "输入指令..."
	cmdInput.Prompt = ":"

	m := &ReaderModel{
		reader:        reader,
		cache:         service.NewPageCache(),
		chapterTitles: chapterTitles,
		curChapter:    0,
		curPage:       0,
		config:        config,
		bookPath:      reader.GetBook().Path,
		cmdInput:      cmdInput,
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
func (m *ReaderModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages (key presses, window resize, async results).
func (m *ReaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (m *ReaderModel) View() string {
	// Show a "parsing…" placeholder until the first chapter is paginated.
	if !m.ready {
		return "正在解析…\n"
	}

	// Popup overlay takes priority.
	if m.showPopup {
		return PopupView(m.popupMsg, m.termWidth, m.termHeight, m.bgColor())
	}

	// Chapter list modal replaces the screen.
	if m.showChapters {
		return ChapterListView(m.chapterTitles, m.chapterCursor, m.termWidth, m.termHeight, m.bgColor())
	}

	// Config settings panel.
	if m.showConfig {
		return ConfigPanelView(&m.config.Settings, m.configCursor, m.termWidth, m.termHeight, m.bgColor())
	}

	// Build the main layout.
	book := m.reader.GetBook()
	chapterTitle := ""
	if m.curChapter < len(m.chapterTitles) {
		chapterTitle = m.chapterTitles[m.curChapter]
	}

	bgColor := m.bgColor()
	header := HeaderView(book.Title, chapterTitle, m.termWidth, bgColor)

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

	body := BodyView(page, m.termWidth, m.termHeight, bgColor)
	footer := FooterView(m.curChapter, m.reader.GetTotalChapters(), m.curPage, m.numPages, m.termWidth, m.commandMode, m.cmdInput.View(), bgColor)

	return header + "\n" + body + "\n" + footer
}

// Cleanup should be called after the Bubble Tea program exits to persist state.
func (m *ReaderModel) Cleanup() {
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
	// Also update library entry with current progress.
	prog := calcProgress(m.curChapter, m.reader.GetTotalChapters(), m.curPage, m.numPages)
	if err := persistence.UpdateBookProgress(m.config, m.bookPath, prog, m.curPage); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: update library progress: %v\n", err)
	}
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

// bgColor returns the current background color based on user settings.
func (m *ReaderModel) bgColor() lipgloss.Color {
	if m.config != nil && m.config.Settings.BgColor {
		return MutedBg
	}
	return lipgloss.Color("")
}

// repaginate triggers async pagination for the current chapter.
func (m *ReaderModel) repaginate() tea.Cmd {
	return paginateChapterCmd(m.reader, m.cache, m.curChapter, m.termWidth, m.termHeight)
}

func (m *ReaderModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// Tier 2: Command mode — enter/esc + textinput.
	if m.commandMode {
		switch msg.String() {
		case "enter":
			return m.executeCommand(m.cmdInput.Value())
		case "esc":
			m.commandMode = false
			m.cmdInput.Reset()
		default:
			var cmd tea.Cmd
			m.cmdInput, cmd = m.cmdInput.Update(msg)
			return m, cmd
		}
	}

	// Tier 3: Chapter list modal mode.
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

	// Tier 4: Config panel mode.
	if m.showConfig {
		switch msg.String() {
		case "up", "k":
			if m.configCursor > 0 {
				m.configCursor--
			}
		case "down", "j":
			if m.configCursor < len(settingsItems)-1 {
				m.configCursor++
			}
		case "enter", " ":
			settingsItems[m.configCursor].set(&m.config.Settings)
			m.configDirty = true
		case "esc", "tab":
			m.showConfig = false
			m.configCursor = 0
			if m.configDirty {
				_ = persistence.SaveSettings(m.config, m.config.Settings)
				m.configDirty = false
			}
		}
		return m, nil
	}

	// Tier 5: Normal reading mode.
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

	case "/":
		m.commandMode = true
		m.cmdInput.Focus()
		m.cmdInput.SetValue("")
		return m, nil
	}

	return m, nil
}

// executeCommand parses and executes a command from the command input.
func (m *ReaderModel) executeCommand(input string) (tea.Model, tea.Cmd) {
	input = strings.TrimSpace(input)
	switch {
	case input == "config":
		m.commandMode = false
		m.cmdInput.Reset()
		if m.config == nil {
			return m, nil
		}
		m.showConfig = true
		m.configCursor = 0
		return m, nil

	case input == "list":
		m.Cleanup()
		return m, func() tea.Msg { return SwitchToLibraryMsg{} }
	case input == "q":
		m.Cleanup()
		return m, tea.Quit
	case strings.HasPrefix(input, "goto "):
		pageStr := strings.TrimPrefix(input, "goto ")
		pageNum, err := strconv.Atoi(strings.TrimSpace(pageStr))
		if err != nil || pageNum < 1 || pageNum > m.numPages {
			m.commandMode = false
			m.cmdInput.Reset()
			return m, nil
		}
		m.curPage = pageNum - 1
		m.commandMode = false
		m.cmdInput.Reset()
		return m, nil
	default:
		m.commandMode = false
		m.cmdInput.Reset()
		return m, nil
	}
}

// gotoChapter jumps to the target chapter, resets page to 0, closes the
// chapter modal, and triggers pagination for the new chapter.
func (m *ReaderModel) gotoChapter(targetChapter int) (tea.Model, tea.Cmd) {
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

func (m *ReaderModel) prevPage() (tea.Model, tea.Cmd) {
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

func (m *ReaderModel) nextPage() (tea.Model, tea.Cmd) {
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

func (m *ReaderModel) prevChapter() (tea.Model, tea.Cmd) {
	if m.curChapter > 0 {
		m.curChapter--
		m.curPage = 0
		return m, m.repaginate()
	}
	return m, nil
}

func (m *ReaderModel) nextChapter() (tea.Model, tea.Cmd) {
	if m.curChapter < m.reader.GetTotalChapters()-1 {
		m.curChapter++
		m.curPage = 0
		return m, m.repaginate()
	}
	return m, nil
}
