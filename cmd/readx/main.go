// readx is a terminal-based novel reader supporting TXT and EPUB formats.
//
// Usage:
//
//	readx [--list] [<book-file>]
//
//	readx                  open library shelf
//	readx --list           open library shelf
//	readx <book-file>      open book directly (auto-adds to library)
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"readx/internal/persistence"
	"readx/internal/ui"
)

func main() {
	flag.Bool("list", false, "open library view")
	flag.Parse()
	config, err := persistence.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		config = &persistence.Config{}
	}

	root := ui.NewRootModel(config)

	hasPath := flag.NArg() > 0
	if hasPath {
		bookPath := flag.Arg(0)
		reader, chapterTitles, cfg, err := ui.OpenBookFile(bookPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer reader.Close()
		config = cfg
		root.OpenBook(reader, chapterTitles, bookPath, cfg)
	} else {
		root.SetState(ui.LibraryState)
	}

	// Save progress on exit (handles ctrl+c which bypasses Update).
	defer root.Cleanup()

	p := tea.NewProgram(root, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
