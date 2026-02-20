package editor

var DefaultColumnWidth = 60

// DisplayLine represents one visual line on screen, mapped back to its source.
type DisplayLine struct {
	BufferLine int    // Index into Buffer.Lines
	Offset     int    // Rune offset within the buffer line where this display line starts
	Text       string // The display text for this line
}

// WrapLine soft-wraps a single hard line into display lines at word boundaries.
// maxWidth is the column width (typically DefaultColumnWidth).
func WrapLine(line string, maxWidth int, bufferLine int) []DisplayLine {
	if maxWidth <= 0 {
		maxWidth = DefaultColumnWidth
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
	Width          int // Terminal width
	Height         int // Terminal height (status bar uses 1 row, so visible = Height-1)
	ColWidth       int // Text column width (capped at TargetColWidth or terminal width)
	LeftMargin     int // Left margin for centring
	TargetColWidth int // User-adjustable target column width
}

func NewViewport(termWidth, termHeight int) *Viewport {
	v := &Viewport{
		Width:          termWidth,
		Height:         termHeight,
		TargetColWidth: DefaultColumnWidth,
	}
	v.recalcLayout()
	return v
}

func (v *Viewport) recalcLayout() {
	target := v.TargetColWidth
	if target <= 0 {
		target = DefaultColumnWidth
	}
	if v.Width >= target {
		v.ColWidth = target
		v.LeftMargin = (v.Width - target) / 2
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
// When at the top of the document (scrollOffset == 0), one line is reserved
// for top padding, giving breathing room from terminal chrome.
func (v *Viewport) VisibleLines(scrollOffset int) int {
	vis := v.Height - 1
	if scrollOffset == 0 && vis > 1 {
		vis--
	}
	return vis
}

// EnsureEndOfFileVisible adjusts scrollOffset to show the end of the file
// when the cursor is near the bottom and the last display line would otherwise
// be hidden below the viewport (e.g. because the last buffer line wraps).
// Only adjusts if the cursor remains visible at the new offset.
func (v *Viewport) EnsureEndOfFileVisible(totalDisplayLines int, cursorDisplayLine int, scrollOffset *int) {
	lastDL := totalDisplayLines - 1
	vis := v.VisibleLines(*scrollOffset)
	if lastDL < *scrollOffset+vis {
		return // Already visible.
	}
	// Scroll down to put lastDL at the bottom. Since we're scrolling down
	// past the initial position, scrollOffset will be > 0, giving us
	// the full Height-1 visible lines.
	newVis := v.Height - 1
	if newVis <= 0 {
		return
	}
	newOffset := lastDL - newVis + 1
	if newOffset < 0 {
		newOffset = 0
		newVis = v.VisibleLines(0)
	}
	// Only apply if cursor remains visible at the new offset.
	if cursorDisplayLine >= newOffset && cursorDisplayLine < newOffset+newVis {
		*scrollOffset = newOffset
	}
}

// EnsureCursorVisible adjusts scrollOffset so the given display line is visible.
func (v *Viewport) EnsureCursorVisible(displayLine int, scrollOffset *int) {
	vis := v.VisibleLines(*scrollOffset)
	if vis <= 0 {
		return
	}
	if displayLine < *scrollOffset {
		*scrollOffset = displayLine
	}
	if displayLine >= *scrollOffset+vis {
		*scrollOffset = displayLine - vis + 1
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
