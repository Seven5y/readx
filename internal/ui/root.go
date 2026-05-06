package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"readx/internal/adapters"
	"readx/internal/domain"
	"readx/internal/persistence"
)

// openBookFile opens a book file and returns the reader, chapter titles, and config.
func OpenBookFile(bookPath string) (domain.Reader, []string, *persistence.Config, error) {
	ext := strings.ToLower(filepath.Ext(bookPath))
	var format domain.Format
	switch ext {
	case ".txt":
		format = domain.FormatTXT
	case ".epub":
		format = domain.FormatEPUB
	default:
		return nil, nil, nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	var reader domain.Reader
	var err error
	switch format {
	case domain.FormatTXT:
		reader, err = adapters.NewTxtAdapter(bookPath)
	case domain.FormatEPUB:
		reader, err = adapters.NewEpubAdapter(bookPath)
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("打开书籍失败: %w", err)
	}

	chapterTitles := getChapterTitles(reader)
	config, err := persistence.LoadConfig()
	if err != nil {
		reader.Close()
		return nil, nil, nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return reader, chapterTitles, config, nil
}

// getChapterTitles extracts titles from a reader efficiently.
func getChapterTitles(reader domain.Reader) []string {
	switch r := reader.(type) {
	case *adapters.TxtAdapter:
		return r.ChapterTitles()
	case *adapters.EpubAdapter:
		return r.ChapterTitles()
	default:
		n := reader.GetTotalChapters()
		titles := make([]string, n)
		for i := 0; i < n; i++ {
			if ch, err := reader.GetChapter(i); err == nil {
				titles[i] = ch.Title
			} else {
				titles[i] = fmt.Sprintf("第%d章", i+1)
			}
		}
		return titles
	}
}


// State represents which view the root model is showing.
type State int

const (
	LibraryState State = iota
	ReaderState
)

// OpenBookMsg is sent when the user selects a book to open from the library.
type OpenBookMsg struct{ Path string }

// SwitchToLibraryMsg is sent when the reader wants to return to the library.
type SwitchToLibraryMsg struct{}

// RootModel is the top-level state machine switching between library and reader.
type RootModel struct {
	state   State
	library *LibraryModel
	reader  *ReaderModel
	config  *persistence.Config
	width   int
	height  int
}

// NewRootModel creates a root model starting in the library state.
func NewRootModel(config *persistence.Config) *RootModel {
	return &RootModel{
		state:   LibraryState,
		library: NewLibraryModel(config),
		config:  config,
	}
}

// SetState forces the root into a specific view.
func (m *RootModel) SetState(s State) { m.state = s }

// OpenBook creates the reader model and switches to reader state.
func (m *RootModel) OpenBook(reader domain.Reader, chapterTitles []string, bookPath string, config *persistence.Config) {
	if err := persistence.AddBook(config, bookPath, reader.GetBook()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: add to library: %v\n", err)
	}
	savedProgress := persistence.GetProgress(config, bookPath)
	m.reader = NewReaderModel(reader, config, chapterTitles, savedProgress)
	m.state = ReaderState
	m.config = config
}

// Cleanup persists reader progress if a reader is active.
func (m *RootModel) Cleanup() {
	if m.reader != nil {
		m.reader.Cleanup()
	}
}

func (m *RootModel) Init() tea.Cmd { return nil }

func (m *RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		var cmd tea.Cmd
		switch m.state {
		case LibraryState:
			if m.library != nil {
				_, cmd = m.library.Update(msg)
			}
		case ReaderState:
			if m.reader != nil {
				_, cmd = m.reader.Update(msg)
			}
		}
		return m, cmd

	case OpenBookMsg:
		reader, chapterTitles, cfg, err := OpenBookFile(msg.Path)
		if err != nil {
			return m, nil
		}
		m.config = cfg
		m.OpenBook(reader, chapterTitles, msg.Path, cfg)
		// Fire a synthetic WindowSizeMsg so the reader paginates with real dimensions.
		if m.width > 0 {
			_, cmd := m.reader.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			return m, cmd
		}
		return m, nil

	case SwitchToLibraryMsg:
		if m.reader != nil {
			m.reader.Cleanup()
			m.reader = nil
		}
		m.library = NewLibraryModel(m.config)
		m.state = LibraryState
		if m.width > 0 {
			_, cmd := m.library.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			return m, cmd
		}
		return m, nil
	}

	var cmd tea.Cmd
	switch m.state {
	case LibraryState:
		if m.library != nil {
			_, cmd = m.library.Update(msg)
		}
	case ReaderState:
		if m.reader != nil {
			_, cmd = m.reader.Update(msg)
		}
	}
	return m, cmd
}

func (m *RootModel) View() string {
	switch m.state {
	case LibraryState:
		if m.library != nil {
			return m.library.View()
		}
	case ReaderState:
		if m.reader != nil {
			return m.reader.View()
		}
	}
	return "loading…"
}
