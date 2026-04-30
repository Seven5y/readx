// Package service provides pagination logic for splitting chapter text
// into screen-sized pages based on terminal dimensions.
package service

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"readx/internal/domain"
)

const (
	headerHeight = 3 // top border + title line + chapter line
	footerHeight = 1 // single footer line
)

// PageCache caches paginated results for current and adjacent chapters
// to avoid holding the entire book's pagination in memory.
type PageCache struct {
	pages map[int][]domain.Page // chapter index → pages
}

// NewPageCache creates a pagination cache.
func NewPageCache() *PageCache {
	return &PageCache{
		pages: make(map[int][]domain.Page),
	}
}

// Get returns the cached pages for a chapter, or nil if not cached.
func (pc *PageCache) Get(chapterIndex int) []domain.Page {
	return pc.pages[chapterIndex]
}

// Set stores paginated pages for a chapter.
func (pc *PageCache) Set(chapterIndex int, pages []domain.Page) {
	pc.pages[chapterIndex] = pages
}

// EvictExcept removes all cached entries except the given indices.
func (pc *PageCache) EvictExcept(keep ...int) {
	keepSet := make(map[int]bool, len(keep))
	for _, k := range keep {
		keepSet[k] = true
	}
	for k := range pc.pages {
		if !keepSet[k] {
			delete(pc.pages, k)
		}
	}
}

// ContentArea returns the width and height available for rendering the
// text content area based on terminal dimensions.
// The -4 accounts for ContentStyle Padding(0,2): 2 left + 2 right columns.
func ContentArea(termWidth, termHeight int) (width, height int) {
	contentWidth := termWidth - 4
	contentHeight := termHeight - headerHeight - footerHeight
	if contentWidth < 20 {
		contentWidth = 20
	}
	if contentHeight < 5 {
		contentHeight = 5
	}
	return contentWidth, contentHeight
}

// Paginate splits a chapter's text into pages that fit within the content area.
// It accounts for CJK character width (2 columns) via go-runewidth.
func Paginate(chapter *domain.Chapter, termWidth, termHeight int) []domain.Page {
	contentWidth, contentHeight := ContentArea(termWidth, termHeight)
	lines := wrapText(chapter.RawContent, contentWidth)

	// Check if all lines are effectively empty.
	allEmpty := true
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			allEmpty = false
			break
		}
	}
	if len(lines) == 0 || allEmpty {
		return []domain.Page{{
			Lines:          []string{"(empty)"},
			ChapterIndex:   chapter.Index,
			PageIndex:      0,
			TotalInChapter: 1,
		}}
	}

	totalPages := (len(lines) + contentHeight - 1) / contentHeight
	pages := make([]domain.Page, 0, totalPages)

	for p := 0; p < totalPages; p++ {
		start := p * contentHeight
		end := start + contentHeight
		if end > len(lines) {
			end = len(lines)
		}

		pages = append(pages, domain.Page{
			Lines:          lines[start:end],
			ChapterIndex:   chapter.Index,
			PageIndex:      p,
			TotalInChapter: totalPages,
		})
	}

	return pages
}

// wrapText wraps text to fit within the given display width, correctly
// accounting for CJK characters that occupy 2 columns.
// Non-empty paragraphs receive a first-line indent (two full-width spaces)
// and consecutive non-empty paragraphs are separated by a blank line.
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return nil
	}

	var lines []string
	paragraphs := strings.Split(text, "\n")

	for _, para := range paragraphs {
		para = strings.TrimRight(para, " \t")
		if para == "" {
			lines = append(lines, "")
			continue
		}

		// First-line indent: two full-width spaces.
		indented := "　　" + para
		lines = append(lines, wrapSingleLine(indented, maxWidth)...)
	}

	return lines
}

// wrapSingleLine wraps a single paragraph (no embedded newlines) to maxWidth.
func wrapSingleLine(text string, maxWidth int) []string {
	var lines []string
	var currentLine strings.Builder
	currentWidth := 0

	for _, r := range text {
		rw := runewidth.RuneWidth(r)

		// If this rune would overflow the line, emit the current line.
		if currentWidth+rw > maxWidth && currentWidth > 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentWidth = 0
		}

		currentLine.WriteRune(r)
		currentWidth += rw
	}

	// Emit any remaining text.
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	// If the text was exactly empty after trimming, emit one empty line.
	if len(lines) == 0 {
		lines = append(lines, "")
	}

	return lines
}

// PaginateOrCache returns pages for a chapter, using the cache if available
// or computing and caching new pages. It also evicts chapters that are not
// current, prev, or next.
func PaginateOrCache(cache *PageCache, reader domain.Reader, chapterIndex int, termWidth, termHeight int) ([]domain.Page, error) {
	if cached := cache.Get(chapterIndex); cached != nil {
		return cached, nil
	}

	ch, err := reader.GetChapter(chapterIndex)
	if err != nil {
		return nil, err
	}

	pages := Paginate(ch, termWidth, termHeight)
	cache.Set(chapterIndex, pages)

	// Pre-fetch adjacent chapters into cache.
	for _, adj := range []int{chapterIndex - 1, chapterIndex + 1} {
		if adj >= 0 && adj < reader.GetTotalChapters() && cache.Get(adj) == nil {
			if adjCh, err := reader.GetChapter(adj); err == nil {
				cache.Set(adj, Paginate(adjCh, termWidth, termHeight))
			}
		}
	}

	// Evict chapters outside the window [current-1, current+1].
	cache.EvictExcept(chapterIndex-1, chapterIndex, chapterIndex+1)

	return pages, nil
}

