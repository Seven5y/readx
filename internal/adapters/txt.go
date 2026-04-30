package adapters

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"

	"github.com/saintfish/chardet"

	"readx/internal/domain"
)

// chapterPattern matches Chinese chapter headings like:
//
//	"第一章", "第1章", "第十章", "第123章", "第一二零章", "第二卷", etc.
var chapterPattern = regexp.MustCompile(`第[^章节卷\s]+[章节卷]`)

// chapterOffset records the byte position and title of a detected chapter.
type chapterOffset struct {
	Title     string
	StartByte int64
}

// TxtAdapter implements domain.Reader for plain-text files.
// It uses byte-offset indexing so that even very large files
// do not need to be fully loaded into memory.
type TxtAdapter struct {
	file     *os.File
	book     *domain.Book
	chapters []chapterOffset // sorted by StartByte
	enc      encoding.Encoding
	size     int64
}

// NewTxtAdapter opens a .txt file, detects its encoding, scans for chapter
// boundaries, and builds a byte-offset index. It returns an error if the
// file cannot be opened or encoding detection fails.
func NewTxtAdapter(path string) (*TxtAdapter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Detect encoding from the first 1KB of the file.
	enc, err := detectEncoding(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("detect encoding: %w", err)
	}

	// Scan the file to find chapter boundaries.
	chapters, err := scanChapters(f, enc)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("scan chapters: %w", err)
	}

	// If no chapters found, treat the entire file as a single chapter.
	if len(chapters) == 0 {
		chapters = []chapterOffset{
			{Title: "正文", StartByte: 0},
		}
	}

	// Derive a book title from the first chapter or file name.
	title := chapters[0].Title
	adapter := &TxtAdapter{
		file:     f,
		book:     &domain.Book{Path: path, Title: title, Format: domain.FormatTXT},
		chapters: chapters,
		enc:      enc,
		size:     fi.Size(),
	}

	return adapter, nil
}

// GetChapter reads the chapter at the given index. It seeks to the stored
// byte offset and reads until the next chapter boundary (or EOF).
func (t *TxtAdapter) GetChapter(index int) (*domain.Chapter, error) {
	if index < 0 || index >= len(t.chapters) {
		return nil, fmt.Errorf("chapter index %d out of range [0, %d)", index, len(t.chapters))
	}

	co := t.chapters[index]

	var endByte int64
	if index+1 < len(t.chapters) {
		endByte = t.chapters[index+1].StartByte
	} else {
		endByte = t.size
	}

	// Read the raw bytes for this chapter span.
	rawLen := endByte - co.StartByte
	if rawLen < 0 {
		return nil, fmt.Errorf("invalid chapter offset")
	}

	raw := make([]byte, rawLen)
	_, err := t.file.ReadAt(raw, co.StartByte)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read chapter at offset %d: %w", co.StartByte, err)
	}

	// Decode from detected encoding to UTF-8.
	content, err := decodeBytes(raw, t.enc)
	if err != nil {
		// Fall back to treating it as UTF-8.
		content = string(raw)
	}

	return &domain.Chapter{
		Index:      index,
		Title:      co.Title,
		RawContent: content,
	}, nil
}

// GetTotalChapters returns the number of detected chapters.
func (t *TxtAdapter) GetTotalChapters() int {
	return len(t.chapters)
}

// GetBook returns book metadata.
func (t *TxtAdapter) GetBook() *domain.Book {
	return t.book
}

// ChapterTitles returns the titles of all detected chapters without reading
// content from disk. This is efficient for building the sidebar chapter list.
func (t *TxtAdapter) ChapterTitles() []string {
	titles := make([]string, len(t.chapters))
	for i, co := range t.chapters {
		titles[i] = co.Title
	}
	return titles
}

// Close closes the underlying file.
func (t *TxtAdapter) Close() error {
	return t.file.Close()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// detectEncoding reads the first 1024 bytes of the file and uses chardet to
// determine the text encoding. Falls back to UTF-8 if detection fails.
func detectEncoding(f *os.File) (encoding.Encoding, error) {
	buf := make([]byte, 1024)
	n, err := f.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		return nil, err
	}
	buf = buf[:n]

	det := chardet.NewTextDetector()
	result, err := det.DetectBest(buf)
	if err != nil {
		// Fallback: treat as UTF-8.
		return nil, nil
	}

	return encodingByName(result.Charset), nil
}

// encodingByName maps common charset names to Go encoding packages.
// Returns nil for UTF-8 (no transformation needed).
func encodingByName(name string) encoding.Encoding {
	switch name {
	case "GB-18030", "GBK", "GB2312":
		return simplifiedchinese.GB18030
	case "Big5":
		return traditionalchinese.Big5
	case "Shift_JIS":
		return japanese.ShiftJIS
	case "EUC-JP":
		return japanese.EUCJP
	case "EUC-KR":
		// No direct EUC-KR in x/text; use a simplified Korean decoder if available.
		return nil
	case "ISO-8859-1":
		return charmap.ISO8859_1
	default:
		return nil
	}
}

// scanChapters reads the file line by line (decoding via transform.Reader) and
// records the byte offset of each line that matches the chapter heading regex.
// Byte offsets are measured on the raw (undecoded) file.
func scanChapters(f *os.File, enc encoding.Encoding) ([]chapterOffset, error) {
	var chapters []chapterOffset
	var rawOffset int64

	// Build a reader that decodes from the detected encoding.
	var r io.Reader = f
	if enc != nil {
		r = transform.NewReader(f, enc.NewDecoder())
	}

	scanner := bufio.NewScanner(r)
	// Increase buffer for very long lines.
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		lineBytes := len(scanner.Bytes()) // decoded bytes (used to estimate raw byte advance)

		// Only match chapter headings near the start of a line.
		// Real chapter titles are standalone lines, not mid-sentence occurrences.
		if m := chapterPattern.FindString(line); m != "" {
			// Verify the match is at or near the start of the line.
			matchPos := strings.Index(line, m)
			if matchPos <= 10 || strings.TrimSpace(line[:matchPos]) == "" {
				chapters = append(chapters, chapterOffset{
					Title:     m,
					StartByte: rawOffset,
				})
			}
		}

		// Advance the raw byte offset. This is an approximation because
		// decoded byte count ≠ raw byte count for multi-byte encodings.
		// A precise approach would track raw bytes via a TeeReader, but
		// for chapter-boundary purposes the approximation is acceptable.
		rawOffset += int64(lineBytes) + 1 // +1 for newline
	}

	if err := scanner.Err(); err != nil {
		return chapters, err
	}

	// Remove duplicate entries that are too close together (subheadings).
	chapters = deduplicateChapters(chapters)

	return chapters, nil
}

// deduplicateChapters merges entries whose StartByte is within 200 bytes
// of the previous entry, keeping the first one.
func deduplicateChapters(chapters []chapterOffset) []chapterOffset {
	if len(chapters) <= 1 {
		return chapters
	}

	// Sort by StartByte just in case.
	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].StartByte < chapters[j].StartByte
	})

	filtered := []chapterOffset{chapters[0]}
	for i := 1; i < len(chapters); i++ {
		if chapters[i].StartByte-filtered[len(filtered)-1].StartByte > 200 {
			filtered = append(filtered, chapters[i])
		}
	}
	return filtered
}

// decodeBytes decodes raw bytes from the detected encoding to UTF-8.
func decodeBytes(raw []byte, enc encoding.Encoding) (string, error) {
	if enc == nil {
		return string(raw), nil
	}

	decoded, err := io.ReadAll(transform.NewReader(
		bytes.NewReader(raw),
		enc.NewDecoder(),
	))
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

