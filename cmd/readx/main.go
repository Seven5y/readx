// readx is a terminal-based novel reader supporting TXT and EPUB formats.
//
// Usage:
//
//	readx <book-file>
//
// Navigation:
//
//	↑/↓ or j/k    page up/down
//	←/→ or h/l    previous/next chapter
//	tab          toggle sidebar
//	q            save progress and quit
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"readx/internal/adapters"
	"readx/internal/domain"
	"readx/internal/persistence"
	"readx/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: readx <book-file>\n")
		os.Exit(1)
	}

	bookPath := os.Args[1]

	// Detect format from file extension.
	ext := strings.ToLower(filepath.Ext(bookPath))
	var format domain.Format
	switch ext {
	case ".txt":
		format = domain.FormatTXT
	case ".epub":
		format = domain.FormatEPUB
	default:
		fmt.Fprintf(os.Stderr, "Unsupported file format: %s\nSupported formats: .txt, .epub\n", ext)
		os.Exit(1)
	}

	// Create the appropriate reader adapter.
	var reader domain.Reader
	var err error

	switch format {
	case domain.FormatTXT:
		reader, err = adapters.NewTxtAdapter(bookPath)
	case domain.FormatEPUB:
		reader, err = adapters.NewEpubAdapter(bookPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening book: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	// Extract chapter titles for the sidebar.
	chapterTitles := getChapterTitles(reader)

	// Load persisted reading progress.
	config, err := persistence.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		config = &persistence.Config{Progress: make(map[string]domain.ReadingProgress)}
	}

	savedProgress := persistence.GetProgress(config, bookPath)

	// Create the UI model.
	model := ui.NewModel(reader, config, chapterTitles, savedProgress)

	// Always save progress on exit (handles ctrl+c which bypasses Update).
	defer model.Cleanup()

	// Run the Bubble Tea program.
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// getChapterTitles extracts chapter titles from the reader using adapter-
// specific efficient methods, falling back to iterating GetChapter.
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
