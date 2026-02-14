package main

import (
	"strings"
	"testing"
)

func TestRenderFrameContainsText(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{
		{BufferLine: 0, Offset: 0, Text: "Hello, world!"},
		{BufferLine: 1, Offset: 0, Text: "Second line."},
	}
	vp := NewViewport(120, 10)

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " test.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	if !strings.Contains(frame, "Hello, world!") {
		t.Error("frame should contain first line text")
	}
	if !strings.Contains(frame, "Second line.") {
		t.Error("frame should contain second line text")
	}
	if !strings.Contains(frame, "test.txt") {
		t.Error("frame should contain filename in status bar")
	}
	if !strings.Contains(frame, "DEFAULT") {
		t.Error("frame should contain mode in status bar")
	}
}

func TestRenderFrameStatusBarReverse(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "text"}}
	vp := NewViewport(80, 5)

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " file.txt", "3 words  EDIT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// Should contain reverse video escape code.
	if !strings.Contains(frame, "\x1b[7m") {
		t.Error("status bar should use reverse video")
	}
	// Should contain reset.
	if !strings.Contains(frame, "\x1b[0m") {
		t.Error("frame should reset attributes after status bar")
	}
}

func TestRenderFrameWithMargin(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "centered"}}
	vp := NewViewport(120, 5) // margin = (120-60)/2 = 30

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// The text should be preceded by spaces for the left margin.
	if !strings.Contains(frame, strings.Repeat(" ", 30)+"centered") {
		t.Error("text should be indented by left margin")
	}
}

func TestRenderFrameScrolled(t *testing.T) {
	r := NewRenderer()
	var dls []DisplayLine
	for i := 0; i < 20; i++ {
		dls = append(dls, DisplayLine{BufferLine: i, Offset: 0, Text: strings.Repeat("x", i+1)})
	}
	vp := NewViewport(120, 10) // 9 visible lines

	frame := r.RenderFrame(dls, vp, 5, 5, 0, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// Line at index 5 has 6 x's. Should be in the frame.
	if !strings.Contains(frame, "xxxxxx") {
		t.Error("should show line at scroll offset")
	}
}

func TestRenderFrameCursorPosition(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "hello"}}
	vp := NewViewport(120, 10) // margin = 10

	frame := r.RenderFrame(dls, vp, 0, 0, 3, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// At scroll 0, top padding = 1. Cursor should be at row 2, col margin+3+1 = 34.
	if !strings.Contains(frame, "\x1b[2;34H") {
		t.Errorf("expected cursor at row 2, col 34. Frame: %q", frame)
	}
}

func TestRenderFrameCursorPositionScrolled(t *testing.T) {
	r := NewRenderer()
	var dls []DisplayLine
	for i := 0; i < 20; i++ {
		dls = append(dls, DisplayLine{BufferLine: i, Offset: 0, Text: "line"})
	}
	vp := NewViewport(120, 10)

	frame := r.RenderFrame(dls, vp, 5, 7, 2, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// screenRow = 7 - 5 + 1 + 0 = 3, screenCol = 30 + 2 + 1 = 33
	if !strings.Contains(frame, "\x1b[3;33H") {
		t.Errorf("expected cursor at row 3, col 33. Frame: %q", frame)
	}
}

func TestRenderFrameTopPadding(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "first line"}}
	vp := NewViewport(80, 5) // No margin (80 < 100)

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " f.txt", "2 words  DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// At scroll 0, content starts at row 2 (top padding = 1).
	if !strings.Contains(frame, "\x1b[2;1H") {
		t.Errorf("content should start at row 2 with top padding. Frame: %q", frame)
	}
}

func TestRenderFrameNoFullClear(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "hello"}}
	vp := NewViewport(80, 5)

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " f.txt", "DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	if strings.Contains(frame, "\x1b[2J") {
		t.Error("frame must not contain full-screen clear (\\x1b[2J)")
	}
	if !strings.Contains(frame, "\x1b[H") {
		t.Error("frame must contain cursor-home (\\x1b[H)")
	}
}

func TestRenderFramePerLineErase(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{
		{BufferLine: 0, Offset: 0, Text: "line one"},
		{BufferLine: 1, Offset: 0, Text: "line two"},
	}
	vp := NewViewport(80, 10)

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " f.txt", "DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// Content lines should be followed by erase-to-end-of-line.
	if !strings.Contains(frame, "line one\x1b[K") {
		t.Error("content line should be followed by \\x1b[K")
	}
	if !strings.Contains(frame, "line two\x1b[K") {
		t.Error("content line should be followed by \\x1b[K")
	}
}

func TestRenderFrameEmptyRowsCleared(t *testing.T) {
	r := NewRenderer()
	// Only 1 content line in a 10-row viewport — empty rows should get \x1b[K.
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "only line"}}
	vp := NewViewport(80, 10)

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " f.txt", "DEFAULT ", PlainHighlighter{}, nil, ModeDefault, -1, -1, false, nil, 0)

	// Count occurrences of erase-to-end-of-line — should appear for every
	// visible row (content + empty viewport rows).
	count := strings.Count(frame, "\x1b[K")
	// At minimum: top padding row + visible content rows + empty rows between content and status bar.
	if count < 2 {
		t.Errorf("expected multiple \\x1b[K sequences for line clearing, got %d", count)
	}
}

func TestRenderPickerOverlay(t *testing.T) {
	r := NewRenderer()
	buffers := []*EditorBuffer{
		NewEditorBuffer("main.go"),
		NewEditorBuffer("utils.go"),
		NewEditorBuffer("README.md"),
	}
	picker := &Picker{Active: true, Selected: 1}
	vp := NewViewport(80, 24)

	result := r.RenderPicker(buffers, picker, 0, vp)

	if !strings.Contains(result, "Open Buffers") {
		t.Error("picker should show title")
	}
	if !strings.Contains(result, "Space-b") {
		t.Error("picker should show keybinding hint")
	}
	if !strings.Contains(result, "main.go") {
		t.Error("picker should show first buffer name")
	}
	if !strings.Contains(result, "utils.go") {
		t.Error("picker should show second buffer name")
	}
	if !strings.Contains(result, "README.md") {
		t.Error("picker should show third buffer name")
	}
	if !strings.Contains(result, ">") {
		t.Error("picker should show selection indicator")
	}
}

func TestApplySpellHighlightingWithOffset(t *testing.T) {
	r := NewRenderer()

	// Simulate a buffer line "hello world test" wrapped into two display lines:
	// Display line 0: "hello world " (offset 0)
	// Display line 1: "test" (offset 12)
	// Spell error on "test" at columns 12-16 in the buffer line.

	dl := DisplayLine{BufferLine: 0, Offset: 12, Text: "test"}
	errors := []SpellError{{Line: 0, StartCol: 12, EndCol: 16, Word: "test"}}

	result := r.applySpellHighlighting(dl.Text, dl, errors)

	// The highlight should start at column 0 of the display line text
	if !strings.Contains(result, "\x1b[38;5;0m\x1b[48;5;224m") {
		t.Error("spell highlight should be applied to wrapped display line")
	}
	// Should contain the reset
	if !strings.Contains(result, "\x1b[39m\x1b[49m") {
		t.Error("spell highlight should be closed")
	}
}

func TestApplySpellHighlightingNoBleed(t *testing.T) {
	r := NewRenderer()

	// Spell error on "world" at columns 6-11 in buffer line.
	// Display line 1 starts at offset 12, so the error should NOT appear here.
	dl := DisplayLine{BufferLine: 0, Offset: 12, Text: "test"}
	errors := []SpellError{{Line: 0, StartCol: 6, EndCol: 11, Word: "world"}}

	result := r.applySpellHighlighting(dl.Text, dl, errors)

	if strings.Contains(result, "\x1b[48;5;224m") {
		t.Error("spell highlight should not bleed onto a different display line")
	}
}

func TestApplySearchHighlightingWithOffset(t *testing.T) {
	r := NewRenderer()

	dl := DisplayLine{BufferLine: 0, Offset: 12, Text: "test"}
	matches := []SearchMatch{{Line: 0, StartCol: 12, EndCol: 16}}

	result := r.applySearchHighlighting(dl.Text, dl, true, matches, 0)

	// Current match highlight (bright yellow)
	if !strings.Contains(result, "\x1b[38;5;0m\x1b[48;5;226m") {
		t.Error("search highlight should be applied to wrapped display line")
	}
	if !strings.Contains(result, "\x1b[39m\x1b[49m") {
		t.Error("search highlight should be closed")
	}
}

func TestApplySearchHighlightingNoBleed(t *testing.T) {
	r := NewRenderer()

	// Search match at columns 0-5 in buffer line, but display line starts at offset 12
	dl := DisplayLine{BufferLine: 0, Offset: 12, Text: "test"}
	matches := []SearchMatch{{Line: 0, StartCol: 0, EndCol: 5}}

	result := r.applySearchHighlighting(dl.Text, dl, true, matches, 0)

	if strings.Contains(result, "\x1b[48;5;226m") || strings.Contains(result, "\x1b[48;5;229m") {
		t.Error("search highlight should not bleed onto a different display line")
	}
}

func TestApplySpellHighlightingFirstDisplayLine(t *testing.T) {
	r := NewRenderer()

	// Error on first display line (offset 0) should still work
	dl := DisplayLine{BufferLine: 0, Offset: 0, Text: "hello world"}
	errors := []SpellError{{Line: 0, StartCol: 6, EndCol: 11, Word: "world"}}

	result := r.applySpellHighlighting(dl.Text, dl, errors)

	if !strings.Contains(result, "\x1b[38;5;0m\x1b[48;5;224m") {
		t.Error("spell highlight should work on first display line")
	}
}

func TestRenderPickerDirtyIndicator(t *testing.T) {
	r := NewRenderer()
	buffers := []*EditorBuffer{
		NewEditorBuffer("clean.go"),
		NewEditorBuffer("dirty.go"),
	}
	buffers[1].buf.Dirty = true
	picker := &Picker{Active: true, Selected: 0}
	vp := NewViewport(80, 24)

	result := r.RenderPicker(buffers, picker, 0, vp)

	// Dirty file should have yellow/bold ANSI code.
	if !strings.Contains(result, "\x1b[1;33m") {
		t.Error("dirty file should be highlighted with yellow/bold")
	}
}
