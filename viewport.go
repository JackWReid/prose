package main

const ColumnWidth = 100

// DisplayLine represents one visual line on screen, mapped back to its source.
type DisplayLine struct {
	BufferLine int    // Index into Buffer.Lines
	Offset     int    // Rune offset within the buffer line where this display line starts
	Text       string // The display text for this line
}

// WrapLine soft-wraps a single hard line into display lines at word boundaries.
// maxWidth is the column width (typically ColumnWidth).
func WrapLine(line string, maxWidth int, bufferLine int) []DisplayLine {
	if maxWidth <= 0 {
		maxWidth = ColumnWidth
	}
	runes := []rune(line)
	if len(runes) == 0 {
		return []DisplayLine{{BufferLine: bufferLine, Offset: 0, Text: ""}}
	}

	var result []DisplayLine
	offset := 0

	for offset < len(runes) {
		remaining := runes[offset:]
		if len(remaining) <= maxWidth {
			result = append(result, DisplayLine{
				BufferLine: bufferLine,
				Offset:     offset,
				Text:       string(remaining),
			})
			break
		}

		// Find the last space within maxWidth characters.
		breakAt := -1
		for i := maxWidth; i > 0; i-- {
			if remaining[i] == ' ' {
				breakAt = i
				break
			}
		}

		if breakAt <= 0 {
			// No word boundary found — hard-break at maxWidth.
			result = append(result, DisplayLine{
				BufferLine: bufferLine,
				Offset:     offset,
				Text:       string(remaining[:maxWidth]),
			})
			offset += maxWidth
		} else {
			result = append(result, DisplayLine{
				BufferLine: bufferLine,
				Offset:     offset,
				Text:       string(remaining[:breakAt]),
			})
			// Skip the space at the break point.
			offset += breakAt + 1
		}
	}

	return result
}

// WrapBuffer wraps all lines in the buffer into display lines.
func WrapBuffer(buf *Buffer, maxWidth int) []DisplayLine {
	var all []DisplayLine
	for i, line := range buf.Lines {
		all = append(all, WrapLine(line, maxWidth, i)...)
	}
	return all
}

// Viewport manages the visible window into the display lines.
type Viewport struct {
	ScrollOffset int // First visible display line index
	Width        int // Terminal width
	Height       int // Terminal height (status bar uses 1 row, so visible = Height-1)
	ColWidth     int // Text column width (capped at ColumnWidth or terminal width)
	LeftMargin   int // Left margin for centring
}

func NewViewport(termWidth, termHeight int) *Viewport {
	v := &Viewport{
		Width:  termWidth,
		Height: termHeight,
	}
	v.recalcLayout()
	return v
}

func (v *Viewport) recalcLayout() {
	if v.Width >= ColumnWidth {
		v.ColWidth = ColumnWidth
		v.LeftMargin = (v.Width - ColumnWidth) / 2
	} else {
		v.ColWidth = v.Width
		v.LeftMargin = 0
	}
}

// Resize updates the viewport for new terminal dimensions.
func (v *Viewport) Resize(termWidth, termHeight int) {
	v.Width = termWidth
	v.Height = termHeight
	v.recalcLayout()
}

// VisibleLines returns the number of text lines visible (excluding status bar).
// When at the top of the document (ScrollOffset == 0), one line is reserved
// for top padding, giving breathing room from terminal chrome.
func (v *Viewport) VisibleLines() int {
	vis := v.Height - 1
	if v.ScrollOffset == 0 && vis > 1 {
		vis--
	}
	return vis
}

// EnsureCursorVisible adjusts ScrollOffset so the given display line is visible.
func (v *Viewport) EnsureCursorVisible(displayLine int) {
	vis := v.VisibleLines()
	if vis <= 0 {
		return
	}
	if displayLine < v.ScrollOffset {
		v.ScrollOffset = displayLine
	}
	if displayLine >= v.ScrollOffset+vis {
		v.ScrollOffset = displayLine - vis + 1
	}
}

// CursorToDisplayLine converts a buffer (line, col) position to a display line
// index and column within the display lines.
func CursorToDisplayLine(displayLines []DisplayLine, bufLine, bufCol int) (displayIdx, displayCol int) {
	for i, dl := range displayLines {
		if dl.BufferLine != bufLine {
			continue
		}
		lineRunes := len([]rune(dl.Text))
		relCol := bufCol - dl.Offset
		// This display line contains the cursor if:
		// - The cursor column is within this segment, OR
		// - This is the last segment of this buffer line and the cursor is at/past the end.
		if relCol >= 0 && relCol <= lineRunes {
			// Check if this is the right segment (not past the end unless last segment).
			isLastSegment := (i+1 >= len(displayLines) || displayLines[i+1].BufferLine != bufLine)
			if relCol < lineRunes || isLastSegment {
				return i, relCol
			}
		}
	}
	// Fallback: put cursor at start of first display line for this buffer line.
	for i, dl := range displayLines {
		if dl.BufferLine == bufLine {
			return i, 0
		}
	}
	return 0, 0
}
