package main

import (
	"fmt"
	"path/filepath"
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
	scrollOffset int,
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

	visibleLines := vp.VisibleLines(scrollOffset)
	topPadding := 0
	if scrollOffset == 0 {
		topPadding = 1
	}
	marginStr := ""
	if vp.LeftMargin > 0 {
		marginStr = strings.Repeat(" ", vp.LeftMargin)
	}

	for i := 0; i < visibleLines; i++ {
		idx := scrollOffset + i
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
	screenRow := cursorDisplayLine - scrollOffset + 1 + topPadding
	screenCol := vp.LeftMargin + cursorDisplayCol + 1
	r.buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", screenRow, screenCol))

	// Show cursor.
	r.buf.WriteString("\x1b[?25h")

	return r.buf.String()
}

// RenderPicker renders the buffer picker overlay centred on screen.
func (r *Renderer) RenderPicker(buffers []*EditorBuffer, picker *Picker, currentBuffer int, vp *Viewport) string {
	var b strings.Builder

	// Hide cursor while picker is shown.
	b.WriteString("\x1b[?25l")

	// Calculate box dimensions.
	maxNameLen := 0
	for _, eb := range buffers {
		name := pickerDisplayName(eb.Filename())
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}
	// Box width: "  > " (4) + name + "  " (2) padding + border (2)
	innerWidth := maxNameLen + 6
	title := " Open buffers "
	if innerWidth < len(title)+2 {
		innerWidth = len(title) + 2
	}
	boxWidth := innerWidth + 2 // +2 for left/right borders
	boxHeight := len(buffers) + 2 // +2 for top/bottom borders

	// Centre the box.
	startCol := (vp.Width - boxWidth) / 2
	if startCol < 0 {
		startCol = 0
	}
	startRow := (vp.Height - boxHeight) / 2
	if startRow < 1 {
		startRow = 1
	}

	// Top border.
	topLine := "┌" + title + strings.Repeat("─", innerWidth-len(title)) + "┐"
	b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", startRow, startCol+1, topLine))

	// Buffer rows.
	for i, eb := range buffers {
		row := startRow + 1 + i
		prefix := "    "
		if i == picker.Selected {
			prefix = "  > "
		}
		name := pickerDisplayName(eb.Filename())

		// Colour dirty filenames yellow/bold.
		displayName := name
		if eb.IsDirty() {
			displayName = "\x1b[1;33m" + name + "\x1b[0m\x1b[7m"
		}

		padding := innerWidth - 4 - len(name)
		if padding < 0 {
			padding = 0
		}
		line := "│" + prefix + displayName + strings.Repeat(" ", padding) + "  │"
		b.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[7m%s\x1b[0m", row, startCol+1, line))
	}

	// Bottom border.
	bottomLine := "└" + strings.Repeat("─", innerWidth) + "┘"
	b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", startRow+boxHeight-1, startCol+1, bottomLine))

	return b.String()
}

func pickerDisplayName(filename string) string {
	if filename == "" {
		return "[unnamed]"
	}
	return filepath.Base(filename)
}

// RenderOutline renders the document outline overlay centred on screen.
func (r *Renderer) RenderOutline(outline *Outline, vp *Viewport) string {
	var b strings.Builder

	// Hide cursor while outline is shown.
	b.WriteString("\x1b[?25l")

	// Max visible items (use ~20 or calculate from viewport).
	maxVisible := 20
	if vp.Height-6 < maxVisible {
		maxVisible = vp.Height - 6
	}
	if maxVisible < 3 {
		maxVisible = 3
	}

	visibleItems := outline.VisibleItems(maxVisible)
	if len(visibleItems) == 0 {
		return b.String()
	}

	// Calculate box dimensions.
	maxTextLen := 0
	for _, item := range visibleItems {
		indent := (item.Level - 1) * 2
		displayLen := indent + len(item.Text)
		if displayLen > maxTextLen {
			maxTextLen = displayLen
		}
	}

	// Box width: "  > " (4) + text + "  " (2) padding + border (2)
	innerWidth := maxTextLen + 6
	title := " Document Outline "
	if innerWidth < len(title)+2 {
		innerWidth = len(title) + 2
	}
	boxWidth := innerWidth + 2 // +2 for left/right borders
	boxHeight := len(visibleItems) + 2 // +2 for top/bottom borders

	// Centre the box.
	startCol := (vp.Width - boxWidth) / 2
	if startCol < 0 {
		startCol = 0
	}
	startRow := (vp.Height - boxHeight) / 2
	if startRow < 1 {
		startRow = 1
	}

	// Top border.
	topLine := "┌" + title + strings.Repeat("─", innerWidth-len(title)) + "┐"
	b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", startRow, startCol+1, topLine))

	// Item rows.
	for i, item := range visibleItems {
		row := startRow + 1 + i
		actualIdx := outline.ScrollOffset + i
		prefix := "    "
		if actualIdx == outline.Selected {
			prefix = "  > "
		}

		// Indent based on heading level.
		indent := strings.Repeat(" ", (item.Level-1)*2)
		displayText := indent + item.Text

		// Calculate padding to fill inner width.
		padding := innerWidth - 4 - len(displayText)
		if padding < 0 {
			// Truncate if too long.
			maxLen := innerWidth - 4
			if maxLen > 0 && len(displayText) > maxLen {
				displayText = displayText[:maxLen]
			}
			padding = 0
		}

		line := "│" + prefix + displayText + strings.Repeat(" ", padding) + "  │"
		b.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[7m%s\x1b[0m", row, startCol+1, line))
	}

	// Bottom border.
	bottomLine := "└" + strings.Repeat("─", innerWidth) + "┘"
	b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", startRow+boxHeight-1, startCol+1, bottomLine))

	// Scroll indicators.
	if outline.ScrollOffset > 0 {
		// Show "↑" indicator at top.
		indicatorRow := startRow + 1
		indicatorCol := startCol + boxWidth - 2
		b.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[7m↑\x1b[0m", indicatorRow, indicatorCol))
	}
	if outline.ScrollOffset+len(visibleItems) < len(outline.Items) {
		// Show "↓" indicator at bottom.
		indicatorRow := startRow + boxHeight - 2
		indicatorCol := startCol + boxWidth - 2
		b.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[7m↓\x1b[0m", indicatorRow, indicatorCol))
	}

	return b.String()
}

func (r *Renderer) renderStatusBar(vp *Viewport, left, right string) {
	row := vp.Height
	r.buf.WriteString(fmt.Sprintf("\x1b[%d;1H", row))
	// Reverse video for status bar.
	r.buf.WriteString("\x1b[7m")

	// Count visible (non-ANSI) characters for layout.
	leftVisible := visibleLen(left)
	rightVisible := visibleLen(right)
	totalWidth := vp.Width

	leftStr := left
	if leftVisible+rightVisible >= totalWidth {
		// Truncate left side if needed.
		maxLeft := totalWidth - rightVisible - 1
		if maxLeft < 0 {
			maxLeft = 0
		}
		leftStr = truncateVisibleStr(left, maxLeft)
		leftVisible = visibleLen(leftStr)
	}

	gap := totalWidth - leftVisible - rightVisible
	if gap < 0 {
		gap = 0
	}

	r.buf.WriteString(leftStr)
	r.buf.WriteString(strings.Repeat(" ", gap))
	r.buf.WriteString(right)

	// Reset attributes.
	r.buf.WriteString("\x1b[0m")
}

// visibleLen counts characters that aren't part of ANSI escape sequences.
func visibleLen(s string) int {
	count := 0
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			i += 2
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				i++
			}
			if i < len(runes) {
				i++
			}
		} else {
			count++
			i++
		}
	}
	return count
}

// truncateVisibleStr truncates a string with ANSI codes to maxVisible visible characters.
func truncateVisibleStr(s string, maxVisible int) string {
	return TruncateVisible(s, maxVisible)
}
