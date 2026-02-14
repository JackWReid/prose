package main

import "testing"

func TestWrapLineShort(t *testing.T) {
	dls := WrapLine("hello world", 100, 0)
	if len(dls) != 1 {
		t.Fatalf("expected 1 display line, got %d", len(dls))
	}
	if dls[0].Text != "hello world" {
		t.Errorf("text: %q", dls[0].Text)
	}
	if dls[0].Offset != 0 {
		t.Errorf("offset: %d", dls[0].Offset)
	}
}

func TestWrapLineEmpty(t *testing.T) {
	dls := WrapLine("", 100, 0)
	if len(dls) != 1 || dls[0].Text != "" {
		t.Errorf("empty line: %v", dls)
	}
}

func TestWrapLineWordBreak(t *testing.T) {
	// Build a line that's exactly 15 chars wide: "aaaa bbbbb cccc"
	// With maxWidth=10, should break at word boundary.
	dls := WrapLine("aaaa bbbbb cccc", 10, 0)
	if len(dls) != 2 {
		t.Fatalf("expected 2 display lines, got %d: %v", len(dls), dls)
	}
	if dls[0].Text != "aaaa bbbbb" {
		t.Errorf("line 0: %q", dls[0].Text)
	}
	if dls[1].Text != "cccc" {
		t.Errorf("line 1: %q", dls[1].Text)
	}
	if dls[1].Offset != 11 {
		t.Errorf("line 1 offset: %d (expected 11)", dls[1].Offset)
	}
}

func TestWrapLineHardBreak(t *testing.T) {
	// A single word longer than maxWidth should be hard-broken.
	dls := WrapLine("abcdefghijklmno", 10, 0)
	if len(dls) != 2 {
		t.Fatalf("expected 2 display lines, got %d", len(dls))
	}
	if dls[0].Text != "abcdefghij" {
		t.Errorf("line 0: %q", dls[0].Text)
	}
	if dls[1].Text != "klmno" {
		t.Errorf("line 1: %q", dls[1].Text)
	}
}

func TestWrapLineMultipleBreaks(t *testing.T) {
	// 30 chars, maxWidth=10. "aaa bbb ccc ddd eee fff ggg"
	line := "aaa bbb ccc ddd eee fff ggg"
	dls := WrapLine(line, 10, 0)
	if len(dls) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(dls))
	}
	for _, dl := range dls {
		if len([]rune(dl.Text)) > 10 {
			t.Errorf("display line exceeds maxWidth: %q (%d)", dl.Text, len([]rune(dl.Text)))
		}
	}
}

func TestWrapBufferMultipleLines(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"short", "also short"}
	dls := WrapBuffer(buf, 100)
	if len(dls) != 2 {
		t.Fatalf("expected 2 display lines, got %d", len(dls))
	}
	if dls[0].BufferLine != 0 || dls[1].BufferLine != 1 {
		t.Errorf("buffer line mapping wrong")
	}
}

func TestWrapBufferWithWrapping(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"aaa bbb ccc", "short"}
	dls := WrapBuffer(buf, 7)
	if len(dls) != 3 {
		t.Fatalf("expected 3 display lines, got %d: %v", len(dls), formatDLs(dls))
	}
}

func TestCursorToDisplayLineSimple(t *testing.T) {
	dls := []DisplayLine{
		{BufferLine: 0, Offset: 0, Text: "hello"},
		{BufferLine: 1, Offset: 0, Text: "world"},
	}

	idx, col := CursorToDisplayLine(dls, 0, 3)
	if idx != 0 || col != 3 {
		t.Errorf("expected (0, 3), got (%d, %d)", idx, col)
	}

	idx, col = CursorToDisplayLine(dls, 1, 0)
	if idx != 1 || col != 0 {
		t.Errorf("expected (1, 0), got (%d, %d)", idx, col)
	}
}

func TestCursorToDisplayLineWrapped(t *testing.T) {
	dls := []DisplayLine{
		{BufferLine: 0, Offset: 0, Text: "aaaa bbbbb"},
		{BufferLine: 0, Offset: 11, Text: "cccc"},
	}

	// Cursor at buffer col 11 should be on display line 1, col 0.
	idx, col := CursorToDisplayLine(dls, 0, 11)
	if idx != 1 || col != 0 {
		t.Errorf("expected (1, 0), got (%d, %d)", idx, col)
	}

	// Cursor at buffer col 13 should be on display line 1, col 2.
	idx, col = CursorToDisplayLine(dls, 0, 13)
	if idx != 1 || col != 2 {
		t.Errorf("expected (1, 2), got (%d, %d)", idx, col)
	}
}

func TestViewportVisibleLines(t *testing.T) {
	vp := NewViewport(120, 10)

	// At top (scrollOffset==0): Height-1 - 1 = 8 (top padding)
	if got := vp.VisibleLines(0); got != 8 {
		t.Errorf("at top: expected 8, got %d", got)
	}

	// When scrolled: Height-1 = 9 (no top padding)
	if got := vp.VisibleLines(1); got != 9 {
		t.Errorf("scrolled: expected 9, got %d", got)
	}
}

func TestViewportVisibleLinesSmallTerminal(t *testing.T) {
	// Height=2 means vis=1; at scroll 0, vis>1 is false so no padding subtracted.
	vp := NewViewport(80, 2)
	if got := vp.VisibleLines(0); got != 1 {
		t.Errorf("small terminal: expected 1, got %d", got)
	}
}

func TestViewportEnsureCursorVisible(t *testing.T) {
	vp := NewViewport(120, 10) // 8 visible lines at top (top padding)
	scrollOffset := 0

	vp.EnsureCursorVisible(0, &scrollOffset)
	if scrollOffset != 0 {
		t.Errorf("scroll should be 0, got %d", scrollOffset)
	}

	// Display line 15 with 8 visible lines at top: scroll to 15-8+1=8
	vp.EnsureCursorVisible(15, &scrollOffset)
	if scrollOffset != 8 {
		t.Errorf("scroll should be 8, got %d", scrollOffset)
	}

	vp.EnsureCursorVisible(5, &scrollOffset)
	if scrollOffset != 5 {
		t.Errorf("scroll should be 5, got %d", scrollOffset)
	}
}

func TestViewportLayoutWide(t *testing.T) {
	vp := NewViewport(200, 50)
	if vp.ColWidth != 60 {
		t.Errorf("col width should be 60, got %d", vp.ColWidth)
	}
	if vp.LeftMargin != 70 {
		t.Errorf("left margin should be 70, got %d", vp.LeftMargin)
	}
}

func TestViewportLayoutNarrow(t *testing.T) {
	vp := NewViewport(60, 20)
	if vp.ColWidth != 60 {
		t.Errorf("col width should be 60, got %d", vp.ColWidth)
	}
	if vp.LeftMargin != 0 {
		t.Errorf("left margin should be 0, got %d", vp.LeftMargin)
	}
}

func TestViewportResize(t *testing.T) {
	vp := NewViewport(200, 50)
	if vp.LeftMargin != 70 {
		t.Fatalf("initial margin: %d", vp.LeftMargin)
	}
	vp.Resize(50, 24)
	if vp.ColWidth != 50 || vp.LeftMargin != 0 {
		t.Errorf("after resize: width=%d, margin=%d", vp.ColWidth, vp.LeftMargin)
	}
}

func formatDLs(dls []DisplayLine) []string {
	var out []string
	for _, dl := range dls {
		out = append(out, dl.Text)
	}
	return out
}
