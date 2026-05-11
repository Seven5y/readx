// Package ui implements the Bubble Tea TUI for the readx terminal reader.
package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

const maxSearchResults = 200

type searchDoneMsg struct {
	query   string
	results []domain.SearchResult
	capped  bool
	err     error
}

type searchFocusType int

const (
	searchFocusInput searchFocusType = iota
	searchFocusList
)

func searchBookCmd(ctx context.Context, reader domain.Reader, chapterTitles []string, query string, termW, termH int, maxResults int) tea.Cmd {
	return func() tea.Msg {
		lowerQuery := strings.ToLower(query)
		totalChapters := reader.GetTotalChapters()
		results := make([]domain.SearchResult, 0)

		for chIdx := 0; chIdx < totalChapters; chIdx++ {
			select {
			case <-ctx.Done():
				return searchDoneMsg{query: query}
			default:
			}

			ch, err := reader.GetChapter(chIdx)
			if err != nil {
				continue
			}
			pages := service.Paginate(ch, termW, termH)

			title := ""
			if chIdx < len(chapterTitles) {
				title = chapterTitles[chIdx]
			}
			for _, page := range pages {
				for lineIdx, line := range page.Lines {
					if strings.Contains(strings.ToLower(line), lowerQuery) {
						results = append(results, domain.SearchResult{
							ChapterIndex: chIdx,
							ChapterTitle: title,
							PageIndex:    page.PageIndex,
							LineIndex:    lineIdx,
							LineContent:  line,
						})
						if len(results) >= maxResults {
							return searchDoneMsg{query: query, results: results, capped: true}
						}
					}
				}
			}
		}

		return searchDoneMsg{query: query, results: results}
	}
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

	showSearch      bool
	searchInput     textinput.Model
	searchResults   []domain.SearchResult
	searchCursor    int
	searchQuery     string
	searchLoading   bool
	searchTruncated bool
	searchFocus     searchFocusType
	searchCancel    context.CancelFunc

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

	searchInput := textinput.New()
	searchInput.Placeholder = "输入关键词..."
	searchInput.Prompt = "搜索: "
	searchInput.TextStyle = lipgloss.NewStyle().Foreground(Primary)
	searchInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(DimText)
	searchInput.PromptStyle = lipgloss.NewStyle().Foreground(Accent)
	searchInput.Cursor.Style = lipgloss.NewStyle().Foreground(Accent)

	m := &ReaderModel{
		reader:        reader,
		cache:         service.NewPageCache(),
		chapterTitles: chapterTitles,
		curChapter:    0,
		curPage:       0,
		config:        config,
		bookPath:      reader.GetBook().Path,
		cmdInput:      cmdInput,
		searchInput: searchInput,
		searchFocus: searchFocusInput,
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

	case searchDoneMsg:
		m.searchLoading = false
		m.searchCancel = nil
		if msg.err != nil {
			return m, nil
		}
		if msg.query == m.searchQuery {
			m.searchResults = msg.results
			m.searchTruncated = msg.capped
			if len(m.searchResults) > 0 {
				m.searchFocus = searchFocusList
				m.searchCursor = 0
			} else {
				m.searchFocus = searchFocusInput
			}
		}
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

	if m.showSearch {
		return SearchView(m.searchInput.View(), m.searchResults, m.searchCursor,
			m.searchLoading, m.searchTruncated, m.termWidth, m.termHeight, m.bgColor())
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

	body := BodyView(page, m.termWidth, m.termHeight, bgColor, m.searchQuery)
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
	// Popup mode.
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

	// Command mode.
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

	// Search mode.
	if m.showSearch {
		switch m.searchFocus {
		case searchFocusInput:
			switch msg.String() {
			case "enter":
				if m.searchLoading {
					return m, nil
				}
				query := strings.TrimSpace(m.searchInput.Value())
				if query == "" {
					return m, nil
				}
				m.searchQuery = query
				m.searchLoading = true
				return m, m.startSearch(query)
			case "esc":
				m.showSearch = false
				m.cancelSearch()
				return m, nil
			case "ctrl+l":
				m.clearSearch()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}

		case searchFocusList:
			switch msg.String() {
			case "up", "k":
				if m.searchCursor > 0 {
					m.searchCursor--
				}
			case "down", "j":
				if m.searchCursor < len(m.searchResults)-1 {
					m.searchCursor++
				}
			case "enter":
				return m.gotoSearchResult(m.searchResults[m.searchCursor])
			case "esc":
				m.showSearch = false
				m.cancelSearch()
				return m, nil
			case "/":
				m.searchFocus = searchFocusInput
				m.searchInput.Focus()
				return m, nil
			case "ctrl+l":
				m.clearSearch()
				m.searchFocus = searchFocusInput
				return m, nil
			}
		}
		return m, nil
	}

	// Chapter list modal mode.
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

	// Config panel mode.
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
		case "left", "h", "right", "l":
			settingsItems[m.configCursor].set(&m.config.Settings)
			m.configDirty = true
		case "enter":
			m.showConfig = false
			m.configCursor = 0
			if m.configDirty {
				_ = persistence.SaveSettings(m.config, m.config.Settings)
				m.configDirty = false
			}
		case "esc", "tab":
			m.showConfig = false
			m.configCursor = 0
			m.configDirty = false
		}
		return m, nil
	}

	// Normal reading mode.
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
	case input == "search":
		m.commandMode = false
		m.cmdInput.Reset()
		m.showSearch = true
		m.searchInput.SetValue(m.searchQuery)
		m.searchInput.Focus()
		m.searchFocus = searchFocusInput
		return m, nil
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

func (m *ReaderModel) clearSearch() {
	m.searchResults = nil
	m.searchQuery = ""
	m.searchCursor = 0
	m.searchTruncated = false
	m.searchInput.Reset()
}

func (m *ReaderModel) cancelSearch() {
	if m.searchCancel != nil {
		m.searchCancel()
		m.searchCancel = nil
	}
	m.searchLoading = false
}

func (m *ReaderModel) startSearch(query string) tea.Cmd {
	m.cancelSearch()
	ctx, cancel := context.WithCancel(context.Background())
	m.searchCancel = cancel
	return searchBookCmd(ctx, m.reader, m.chapterTitles, query, m.termWidth, m.termHeight, maxSearchResults)
}

func (m *ReaderModel) gotoSearchResult(result domain.SearchResult) (tea.Model, tea.Cmd) {
	m.showSearch = false
	if result.ChapterIndex != m.curChapter {
		m.curChapter = result.ChapterIndex
		m.curPage = result.PageIndex
		return m, m.repaginate()
	}
	m.curPage = result.PageIndex
	return m, nil
}

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
