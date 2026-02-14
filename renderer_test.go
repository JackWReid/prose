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

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " test.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil)

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

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " file.txt", "3 words  EDIT ", PlainHighlighter{}, nil)

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
	vp := NewViewport(120, 5) // margin = (120-100)/2 = 10

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil)

	// The text should be preceded by spaces for the left margin.
	if !strings.Contains(frame, strings.Repeat(" ", 10)+"centered") {
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

	frame := r.RenderFrame(dls, vp, 5, 5, 0, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil)

	// Line at index 5 has 6 x's. Should be in the frame.
	if !strings.Contains(frame, "xxxxxx") {
		t.Error("should show line at scroll offset")
	}
}

func TestRenderFrameCursorPosition(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "hello"}}
	vp := NewViewport(120, 10) // margin = 10

	frame := r.RenderFrame(dls, vp, 0, 0, 3, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil)

	// At scroll 0, top padding = 1. Cursor should be at row 2, col margin+3+1 = 14.
	if !strings.Contains(frame, "\x1b[2;14H") {
		t.Errorf("expected cursor at row 2, col 14. Frame: %q", frame)
	}
}

func TestRenderFrameCursorPositionScrolled(t *testing.T) {
	r := NewRenderer()
	var dls []DisplayLine
	for i := 0; i < 20; i++ {
		dls = append(dls, DisplayLine{BufferLine: i, Offset: 0, Text: "line"})
	}
	vp := NewViewport(120, 10)

	frame := r.RenderFrame(dls, vp, 5, 7, 2, " f.txt", "5 words  DEFAULT ", PlainHighlighter{}, nil)

	// screenRow = 7 - 5 + 1 + 0 = 3, screenCol = 10 + 2 + 1 = 13
	if !strings.Contains(frame, "\x1b[3;13H") {
		t.Errorf("expected cursor at row 3, col 13. Frame: %q", frame)
	}
}

func TestRenderFrameTopPadding(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "first line"}}
	vp := NewViewport(80, 5) // No margin (80 < 100)

	frame := r.RenderFrame(dls, vp, 0, 0, 0, " f.txt", "2 words  DEFAULT ", PlainHighlighter{}, nil)

	// At scroll 0, content starts at row 2 (top padding = 1).
	if !strings.Contains(frame, "\x1b[2;1H") {
		t.Errorf("content should start at row 2 with top padding. Frame: %q", frame)
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
