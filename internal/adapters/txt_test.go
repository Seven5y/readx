package adapters

import (
	"testing"
)

func TestChapterPattern_ChineseNumbers(t *testing.T) {
	tests := []struct {
		input string
		match string
	}{
		{"第一章 风云再起", "第一章"},
		{"第十章 大结局", "第十章"},
		{"第123章 新的开始", "第123章"},
		{"第一二零章 回府", "第一二零章"},
		{"第一卷 初入江湖", "第一卷"},
		{"第三卷", "第三卷"},
		{"第廿三章 秘闻", "第廿三章"},
	}

	for _, tt := range tests {
		got := chapterPattern.FindString(tt.input)
		if got != tt.match {
			t.Errorf("chapterPattern.FindString(%q) = %q, want %q", tt.input, got, tt.match)
		}
	}
}

// TestChapterPattern_FalsePositives verifies that the regex can produce
// false positives on mid-sentence matches, which is expected behavior.
// The actual filtering (position check) happens in scanChapters.
func TestChapterPattern_FalsePositives(t *testing.T) {
	// The regex alone matches these, but scanChapters filters them by position.
	input := "这是第三章的内容介绍，但不是标题。"
	got := chapterPattern.FindString(input)
	if got == "" {
		t.Errorf("regex should find '第三章' even mid-sentence (filtered by scanChapters)")
	}
}

func TestDeduplicateChapters(t *testing.T) {
	chapters := []chapterOffset{
		{Title: "第一章", StartByte: 0},
		{Title: "第一节", StartByte: 50},       // too close to first, should be removed
		{Title: "第二章", StartByte: 1000},      // far enough
		{Title: "第二节", StartByte: 1050},      // too close to "第二章"
		{Title: "第三章", StartByte: 3000},
	}

	result := deduplicateChapters(chapters)

	if len(result) != 3 {
		t.Fatalf("expected 3 chapters after dedup, got %d: %v", len(result), result)
	}
	if result[0].Title != "第一章" {
		t.Errorf("first chapter = %q", result[0].Title)
	}
	if result[1].Title != "第二章" {
		t.Errorf("second chapter = %q", result[1].Title)
	}
	if result[2].Title != "第三章" {
		t.Errorf("third chapter = %q", result[2].Title)
	}
}
