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

	frame := r.RenderFrame(dls, vp, 0, 0, " test.txt", "DEFAULT ")

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

	frame := r.RenderFrame(dls, vp, 0, 0, " file.txt [+]", "EDIT ")

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

	frame := r.RenderFrame(dls, vp, 0, 0, " f.txt", "DEFAULT ")

	// The text should be preceded by spaces for the left margin.
	// After the cursor positioning escape, we should find spaces before "centered".
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
	vp.ScrollOffset = 5

	frame := r.RenderFrame(dls, vp, 5, 0, " f.txt", "DEFAULT ")

	// Line at index 5 has 6 x's. Should be in the frame.
	if !strings.Contains(frame, "xxxxxx") {
		t.Error("should show line at scroll offset")
	}
	// Line at index 0 should NOT be visible (it has 1 'x').
	// This is tricky to test precisely, but line 0's unique text is a single 'x',
	// which appears as a substring of others. So skip negative assertion.
}

func TestRenderFrameCursorPosition(t *testing.T) {
	r := NewRenderer()
	dls := []DisplayLine{{BufferLine: 0, Offset: 0, Text: "hello"}}
	vp := NewViewport(120, 10) // margin = 10

	frame := r.RenderFrame(dls, vp, 0, 3, " f.txt", "DEFAULT ")

	// Cursor should be at row 1, col margin+3+1 = 14.
	if !strings.Contains(frame, "\x1b[1;14H") {
		t.Errorf("expected cursor at row 1, col 14. Frame: %q", frame)
	}
}
