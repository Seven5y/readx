package service

import (
	"strings"
	"testing"

	"readx/internal/domain"
)

func TestWrapText_CJKWidth(t *testing.T) {
	// "你好世界" = 4 CJK chars, each 2 columns wide = 8 column width.
	text := "你好世界"
	lines := wrapText(text, 6)

	// With maxWidth=6, the first line fits 3 CJK chars (6 columns),
	// the second line gets the remaining 1 CJK char.
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "你好世" {
		t.Errorf("first line = %q, want %q", lines[0], "你好世")
	}
	if lines[1] != "界" {
		t.Errorf("second line = %q, want %q", lines[1], "界")
	}
}

func TestWrapText_MixedLatinCJK(t *testing.T) {
	// "A你B好" = A(1) + 你(2) + B(1) + 好(2) = 6 columns.
	text := "A你B好C世"
	lines := wrapText(text, 6)

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "A你B好" {
		t.Errorf("first line = %q, want %q", lines[0], "A你B好")
	}
	if lines[1] != "C世" {
		t.Errorf("second line = %q, want %q", lines[1], "C世")
	}
}

func TestWrapText_PreservesBlankLines(t *testing.T) {
	text := "line one\n\nline three"
	lines := wrapText(text, 80)

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (with blank middle), got %d: %v", len(lines), lines)
	}
	if lines[0] != "line one" {
		t.Errorf("lines[0] = %q", lines[0])
	}
	if lines[1] != "" {
		t.Errorf("lines[1] (blank) = %q", lines[1])
	}
	if lines[2] != "line three" {
		t.Errorf("lines[2] = %q", lines[2])
	}
}

func TestWrapText_ShortLine(t *testing.T) {
	text := "short"
	lines := wrapText(text, 80)
	if len(lines) != 1 || lines[0] != "short" {
		t.Errorf("expected single line %q, got %v", "short", lines)
	}
}

func TestPaginate_EmptyChapter(t *testing.T) {
	ch := &domain.Chapter{Index: 0, Title: "Empty", RawContent: ""}
	pages := Paginate(ch, 80, 24)

	if len(pages) != 1 {
		t.Fatalf("expected 1 page (empty placeholder), got %d", len(pages))
	}
	if !strings.Contains(pages[0].Lines[0], "(empty)") {
		t.Errorf("expected '(empty)' placeholder, got %q", pages[0].Lines[0])
	}
}

func TestPaginate_PageCount(t *testing.T) {
	// With content area height around 20 (24 - 3 - 1), 50 lines → 3 pages.
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "x")
	}
	text := strings.Join(lines, "\n")

	ch := &domain.Chapter{Index: 0, Title: "Test", RawContent: text}
	pages := Paginate(ch, 80, 24)

	// contentHeight = 24 - 3 - 1 = 20, so 50 lines ÷ 20 = 3 pages.
	if len(pages) != 3 {
		t.Errorf("expected 3 pages (50 lines / 20 per page), got %d", len(pages))
	}
	if pages[0].PageIndex != 0 || pages[1].PageIndex != 1 || pages[2].PageIndex != 2 {
		t.Errorf("page indices wrong: %v", pages)
	}
	if pages[0].TotalInChapter != 3 {
		t.Errorf("TotalInChapter = %d, want 3", pages[0].TotalInChapter)
	}
}

func TestContentArea(t *testing.T) {
	w, h := ContentArea(100, 30)
	// width = 100 * 0.8 = 80, height = 30 - 3 - 1 = 26
	if w != 80 {
		t.Errorf("width = %d, want 80", w)
	}
	if h != 26 {
		t.Errorf("height = %d, want 26", h)
	}
}

func TestContentArea_Minimum(t *testing.T) {
	w, h := ContentArea(10, 5)
	if w < 20 {
		t.Errorf("width should be clamped to 20, got %d", w)
	}
	if h < 5 {
		t.Errorf("height should be clamped to 5, got %d", h)
	}
}

func TestPageCache_EvictExcept(t *testing.T) {
	cache := NewPageCache()
	cache.Set(0, []domain.Page{{PageIndex: 0}})
	cache.Set(1, []domain.Page{{PageIndex: 1}})
	cache.Set(2, []domain.Page{{PageIndex: 2}})
	cache.Set(3, []domain.Page{{PageIndex: 3}})

	cache.EvictExcept(1, 2)

	if cache.Get(0) != nil {
		t.Error("chapter 0 should have been evicted")
	}
	if cache.Get(3) != nil {
		t.Error("chapter 3 should have been evicted")
	}
	if cache.Get(1) == nil {
		t.Error("chapter 1 should still be cached")
	}
	if cache.Get(2) == nil {
		t.Error("chapter 2 should still be cached")
	}
}
