package domain

// Reader is the abstraction for reading books in different formats.
// Each format (txt, epub) provides its own implementation.
type Reader interface {
	// GetChapter returns the chapter at the given 0-based index.
	GetChapter(index int) (*Chapter, error)

	// GetTotalChapters returns the number of chapters in the book.
	GetTotalChapters() int

	// GetBook returns the book metadata.
	GetBook() *Book

	// Close releases any resources held by the reader.
	Close() error
}
