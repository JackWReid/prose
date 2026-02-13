package main

import (
	"fmt"
	"strings"
)

// Renderer builds a frame buffer and writes it to the terminal in one go.
type Renderer struct {
	buf strings.Builder
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

// RenderFrame draws the full screen: text lines + status bar + cursor placement.
func (r *Renderer) RenderFrame(
	displayLines []DisplayLine,
	vp *Viewport,
	cursorDisplayLine int,
	cursorDisplayCol int,
	statusLeft string,
	statusRight string,
	highlighter Highlighter,
) string {
	r.buf.Reset()

	// Hide cursor during drawing.
	r.buf.WriteString("\x1b[?25l")

	// Clear screen and move to top-left.
	r.buf.WriteString("\x1b[2J\x1b[H")

	visibleLines := vp.VisibleLines()
	topPadding := 0
	if vp.ScrollOffset == 0 {
		topPadding = 1
	}
	marginStr := ""
	if vp.LeftMargin > 0 {
		marginStr = strings.Repeat(" ", vp.LeftMargin)
	}

	for i := 0; i < visibleLines; i++ {
		idx := vp.ScrollOffset + i
		// Move to row (1-indexed), offset by top padding.
		row := i + 1 + topPadding
		r.buf.WriteString(fmt.Sprintf("\x1b[%d;1H", row))
		if idx < len(displayLines) {
			text := displayLines[idx].Text
			text = highlighter.Highlight(text)
			text = TruncateVisible(text, vp.ColWidth)
			r.buf.WriteString(marginStr)
			r.buf.WriteString(text)
		}
	}

	// Status bar on the last row.
	r.renderStatusBar(vp, statusLeft, statusRight)

	// Position the cursor.
	screenRow := cursorDisplayLine - vp.ScrollOffset + 1 + topPadding
	screenCol := vp.LeftMargin + cursorDisplayCol + 1
	r.buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", screenRow, screenCol))

	// Show cursor.
	r.buf.WriteString("\x1b[?25h")

	return r.buf.String()
}

func (r *Renderer) renderStatusBar(vp *Viewport, left, right string) {
	row := vp.Height
	r.buf.WriteString(fmt.Sprintf("\x1b[%d;1H", row))
	// Reverse video for status bar.
	r.buf.WriteString("\x1b[7m")

	leftRunes := []rune(left)
	rightRunes := []rune(right)
	totalWidth := vp.Width

	if len(leftRunes)+len(rightRunes) >= totalWidth {
		// Truncate left side if needed.
		maxLeft := totalWidth - len(rightRunes) - 1
		if maxLeft < 0 {
			maxLeft = 0
		}
		if len(leftRunes) > maxLeft {
			leftRunes = leftRunes[:maxLeft]
		}
	}

	gap := totalWidth - len(leftRunes) - len(rightRunes)
	if gap < 0 {
		gap = 0
	}

	r.buf.WriteString(string(leftRunes))
	r.buf.WriteString(strings.Repeat(" ", gap))
	r.buf.WriteString(string(rightRunes))

	// Reset attributes.
	r.buf.WriteString("\x1b[0m")
}
