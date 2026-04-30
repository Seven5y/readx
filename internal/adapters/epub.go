package adapters

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"readx/internal/domain"
)

// EpubAdapter implements domain.Reader for EPUB files.
// EPUB is a ZIP archive containing XHTML files organized by a manifest and spine.
type EpubAdapter struct {
	book     *domain.Book
	chapters []domain.Chapter
}

// NewEpubAdapter opens an .epub file, parses its structure, and extracts all
// chapters into memory. EPUB files are typically small enough for in-memory storage.
func NewEpubAdapter(path string) (*EpubAdapter, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open epub: %w", err)
	}
	defer zr.Close()

	// Locate the OPF file via container.xml.
	opfPath, err := findOPFPath(zr)
	if err != nil {
		return nil, fmt.Errorf("find OPF: %w", err)
	}

	// Parse the OPF file for metadata and spine.
	opfDir := filepath.Dir(opfPath)
	opf, err := parseOPF(zr, opfPath)
	if err != nil {
		return nil, fmt.Errorf("parse OPF: %w", err)
	}

	// Extract each spine item's content.
	chapters := make([]domain.Chapter, 0, len(opf.Spine))
	for i, itemRef := range opf.Spine {
		href := opf.Manifest[itemRef.IDREF]
		if href == "" {
			continue
		}
		fullPath := resolvePath(opfDir, href)

		text, err := extractHTMLText(zr, fullPath)
		if err != nil {
			// Skip broken items rather than failing the whole book.
			text = "(unable to read chapter content)"
		}

		// Detect chapter title from content or use a fallback.
		title := detectChapterTitle(text)
		if title == "" {
			title = fmt.Sprintf("第%d章", i+1)
		}

		chapters = append(chapters, domain.Chapter{
			Index:      i,
			Title:      title,
			RawContent: text,
		})
	}

	// If the spine had no readable items, create a single fallback chapter.
	if len(chapters) == 0 {
		chapters = append(chapters, domain.Chapter{
			Index:      0,
			Title:      "正文",
			RawContent: "(empty book)",
		})
	}

	return &EpubAdapter{
		book: &domain.Book{
			Path:   path,
			Title:  opf.Title,
			Author: opf.Creator,
			Format: domain.FormatEPUB,
		},
		chapters: chapters,
	}, nil
}

// GetChapter returns a pointer to the chapter at the given index.
func (e *EpubAdapter) GetChapter(index int) (*domain.Chapter, error) {
	if index < 0 || index >= len(e.chapters) {
		return nil, fmt.Errorf("chapter index %d out of range [0, %d)", index, len(e.chapters))
	}
	ch := e.chapters[index]
	return &ch, nil
}

// GetTotalChapters returns the number of chapters.
func (e *EpubAdapter) GetTotalChapters() int {
	return len(e.chapters)
}

// GetBook returns book metadata.
func (e *EpubAdapter) GetBook() *domain.Book {
	return e.book
}

// ChapterTitles returns the titles of all chapters.
func (e *EpubAdapter) ChapterTitles() []string {
	titles := make([]string, len(e.chapters))
	for i, ch := range e.chapters {
		titles[i] = ch.Title
	}
	return titles
}

// Close is a no-op for EpubAdapter (all data is in memory).
func (e *EpubAdapter) Close() error {
	return nil
}

// ---------------------------------------------------------------------------
// EPUB parsing types and helpers
// ---------------------------------------------------------------------------

// opfData holds the relevant fields parsed from the OPF (content.opf) file.
type opfData struct {
	Title    string
	Creator  string
	Manifest map[string]string // id → href
	Spine    []struct{ IDREF string }
}

// containerXML is the minimal structure for parsing META-INF/container.xml.
type containerXML struct {
	RootFiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

// findOPFPath reads container.xml from the EPUB ZIP and returns the OPF path.
func findOPFPath(zr *zip.ReadCloser) (string, error) {
	f, err := zr.Open("META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("open container.xml: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("read container.xml: %w", err)
	}

	var c containerXML
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}

	if len(c.RootFiles) == 0 {
		return "", fmt.Errorf("no rootfile in container.xml")
	}

	return c.RootFiles[0].FullPath, nil
}

// parseOPF reads and parses the OPF file from the ZIP.
func parseOPF(zr *zip.ReadCloser, opfPath string) (*opfData, error) {
	f, err := zr.Open(opfPath)
	if err != nil {
		return nil, fmt.Errorf("open OPF %s: %w", opfPath, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read OPF: %w", err)
	}

	// OPF namespace is usually http://www.idpf.org/2007/opf, but we parse
	// generically to avoid namespace complexity.
	var raw struct {
		Metadata struct {
			Titles  []string `xml:"title"`
			Creator string   `xml:"creator"`
		} `xml:"metadata"`
		Manifest struct {
			Items []struct {
				ID   string `xml:"id,attr"`
				Href string `xml:"href,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
		Spine struct {
			Items []struct {
				IDREF string `xml:"idref,attr"`
			} `xml:"itemref"`
		} `xml:"spine"`
	}

	if err := xml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse OPF XML: %w", err)
	}

	opf := &opfData{
		Manifest: make(map[string]string),
	}
	if len(raw.Metadata.Titles) > 0 {
		opf.Title = raw.Metadata.Titles[0]
	}
	opf.Creator = raw.Metadata.Creator

	for _, item := range raw.Manifest.Items {
		opf.Manifest[item.ID] = item.Href
	}
	for _, itemRef := range raw.Spine.Items {
		opf.Spine = append(opf.Spine, struct{ IDREF string }{IDREF: itemRef.IDREF})
	}

	return opf, nil
}

// Patterns used by extractHTMLText.
var (
	tagStripRe  = regexp.MustCompile(`<[^>]*>`)
	brRe        = regexp.MustCompile(`<br\s*/?>`)
	blockCloseRe = regexp.MustCompile(`</?(?:p|div|h[1-6]|tr|li|blockquote|section|article|pre|hr)\s*/?>`)
)

// extractHTMLText reads an HTML file from the ZIP and returns plain text
// with paragraph structure preserved for the pagination beautification.
func extractHTMLText(zr *zip.ReadCloser, path string) (string, error) {
	f, err := zr.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	text := string(data)

	// Convert block-level tags to newlines to preserve paragraph boundaries.
	text = brRe.ReplaceAllString(text, "\n")
	text = blockCloseRe.ReplaceAllString(text, "\n")

	// Strip remaining inline tags (replace with space to avoid word concatenation).
	text = tagStripRe.ReplaceAllString(text, " ")

	// Decode HTML entities.
	text = html.UnescapeString(text)

	// Collapse whitespace per-line, preserving blank lines as paragraph separators.
	rawLines := strings.Split(text, "\n")
	var lines []string
	for _, line := range rawLines {
		line = strings.Join(strings.Fields(line), " ")
		lines = append(lines, line)
	}
	text = strings.Join(lines, "\n")
	text = strings.TrimSpace(text)

	return text, nil
}

// detectChapterTitle attempts to extract a chapter title from the content.
func detectChapterTitle(content string) string {
	match := chapterPattern.FindString(content)
	return match
}

// resolvePath resolves a relative href against the OPF directory.
func resolvePath(opfDir, href string) string {
	if strings.HasPrefix(href, "/") {
		return strings.TrimPrefix(href, "/")
	}
	return filepath.Clean(filepath.Join(opfDir, href))
}
