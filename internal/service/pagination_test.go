package service

import (
	"strings"
	"testing"

	"readx/internal/domain"
)

func TestWrapText_CJKWidth(t *testing.T) {
	// "你好世界" = 4 CJK chars, each 2 columns wide = 8 columns.
	// With indent "　　你好世界" = 12 columns. Wrapped at 6 → ["　　你", "好世界"].
	text := "你好世界"
	lines := wrapText(text, 6)

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "　　你" {
		t.Errorf("first line = %q, want %q", lines[0], "　　你")
	}
	if lines[1] != "好世界" {
		t.Errorf("second line = %q, want %q", lines[1], "好世界")
	}
}

func TestWrapText_MixedLatinCJK(t *testing.T) {
	// "A你B好C世" with indent "　　　A你B好C世" = 4+1+2+1+2+1+2 = 13 columns.
	// Wrapped at 6 → ["　　A", "你B好C", "世"].
	text := "A你B好C世"
	lines := wrapText(text, 6)

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "　　A" {
		t.Errorf("first line = %q, want %q", lines[0], "　　A")
	}
	if lines[1] != "你B好C" {
		t.Errorf("second line = %q, want %q", lines[1], "你B好C")
	}
	if lines[2] != "世" {
		t.Errorf("third line = %q, want %q", lines[2], "世")
	}
}

func TestWrapText_PreservesBlankLines(t *testing.T) {
	text := "line one\n\nline three"
	lines := wrapText(text, 80)

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (with blank middle), got %d: %v", len(lines), lines)
	}
	if lines[0] != "　　line one" {
		t.Errorf("lines[0] = %q, want %q", lines[0], "　　line one")
	}
	if lines[1] != "" {
		t.Errorf("lines[1] (blank) = %q", lines[1])
	}
	if lines[2] != "　　line three" {
		t.Errorf("lines[2] = %q, want %q", lines[2], "　　line three")
	}
}

func TestWrapText_ShortLine(t *testing.T) {
	text := "short"
	lines := wrapText(text, 80)
	if len(lines) != 1 || lines[0] != "　　short" {
		t.Errorf("expected single line %q, got %v", "　　short", lines)
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
	// 50 paragraphs → 50 indented lines (no inter-paragraph separator).
	// contentHeight = 24 - 3 - 1 = 20, so 50 lines ÷ 20 = 3 pages.
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "x")
	}
	text := strings.Join(lines, "\n")

	ch := &domain.Chapter{Index: 0, Title: "Test", RawContent: text}
	pages := Paginate(ch, 80, 24)

	if len(pages) != 3 {
		t.Errorf("expected 3 pages (50 lines / 20 per page), got %d", len(pages))
	}
	if pages[0].TotalInChapter != 3 {
		t.Errorf("TotalInChapter = %d, want 3", pages[0].TotalInChapter)
	}
}

func TestContentArea(t *testing.T) {
	// width = 100 - 4 = 96, height = 30 - 3 - 1 = 26.
	w, h := ContentArea(100, 30)
	if w != 96 {
		t.Errorf("width = %d, want 96", w)
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
