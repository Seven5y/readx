// Package domain defines core data types and interfaces for the readx terminal reader.
package domain

import "time"

// Format represents the file format of a book.
type Format string

const (
	FormatTXT  Format = "txt"
	FormatEPUB Format = "epub"
)

// Book holds metadata about a book file.
type Book struct {
	Path   string
	Title  string
	Author string
	Format Format
}

// Chapter represents a single chapter within a book.
type Chapter struct {
	Index      int    // 0-based chapter index
	Title      string // chapter title (e.g. "第一章 风云再起")
	RawContent string // plain text content, all HTML tags stripped
}

// Page represents a single screen of text after pagination.
type Page struct {
	Lines         []string // the text lines that fit on this page
	ChapterIndex  int      // which chapter this page belongs to
	PageIndex     int      // page index within the chapter (0-based)
	TotalInChapter int     // total pages in this chapter
}

// ReadingProgress records the last reading position for a book.
type ReadingProgress struct {
	BookPath     string    `json:"book_path"`
	ChapterIndex int       `json:"chapter_index"`
	PageIndex    int       `json:"page_index"`
	Timestamp    time.Time `json:"timestamp"`
}

// LibraryEntry holds metadata for a book in the library shelf.
type LibraryEntry struct {
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	Author   string    `json:"author"`
	Format   Format    `json:"format"`
	Progress int       `json:"progress"`  // 0-100
	LastPage int       `json:"last_page"` // page index
	LastRead time.Time `json:"last_read"`
}
