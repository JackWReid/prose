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
	spellErrors []SpellError,
	mode Mode,
	selectionStart int,
	selectionEnd int,
	searchActive bool,
	searchMatches []SearchMatch,
	searchCurrentIdx int,
) string {
	r.buf.Reset()

	// Hide cursor during drawing.
	r.buf.WriteString("\x1b[?25l")

	// Move cursor to top-left (no full-screen clear — we erase per line instead).
	r.buf.WriteString("\x1b[H")

	visibleLines := vp.VisibleLines(scrollOffset)
	topPadding := 0
	if scrollOffset == 0 {
		topPadding = 1
	}
	marginStr := ""
	if vp.LeftMargin > 0 {
		marginStr = strings.Repeat(" ", vp.LeftMargin)
	}

	// Clear top padding row if present.
	if topPadding > 0 {
		r.buf.WriteString("\x1b[1;1H\x1b[K")
	}

	for i := 0; i < visibleLines; i++ {
		idx := scrollOffset + i
		// Move to row (1-indexed), offset by top padding.
		row := i + 1 + topPadding
		r.buf.WriteString(fmt.Sprintf("\x1b[%d;1H", row))
		if idx < len(displayLines) {
			text := displayLines[idx].Text
			text = highlighter.Highlight(text)
			text = r.applySpellHighlighting(text, displayLines[idx], spellErrors)
			text = r.applySearchHighlighting(text, displayLines[idx], searchActive, searchMatches, searchCurrentIdx)
			text = TruncateVisible(text, vp.ColWidth)

			// Apply reverse video for line-select mode
			if mode == ModeLineSelect {
				bufLine := displayLines[idx].BufferLine
				if bufLine >= selectionStart && bufLine <= selectionEnd {
					text = "\x1b[7m" + text + "\x1b[0m"
				}
			}

			r.buf.WriteString(marginStr)
			r.buf.WriteString(text)
		}
		// Erase to end of line (clears stale content without a full-screen clear).
		r.buf.WriteString("\x1b[K")
	}

	// Clear any remaining rows between content and status bar.
	lastContentRow := visibleLines + topPadding
	statusRow := vp.Height
	for row := lastContentRow + 1; row < statusRow; row++ {
		r.buf.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[K", row))
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
	// Build items for overlay.
	items := make([]OverlayItem, len(buffers))
	for i, eb := range buffers {
		name := pickerDisplayName(eb.Filename(), eb.isScratch)
		displayName := name
		// Colour dirty filenames yellow/bold.
		if eb.IsDirty() {
			displayName = "\x1b[1;33m" + name + "\x1b[0m"
		}
		items[i] = OverlayItem{
			DisplayText: displayName,
			RawText:     name,
		}
	}

	return r.RenderOverlay(
		"Open Buffers",
		"Space-b/t",
		items,
		picker.Selected,
		vp,
		OverlayScrollInfo{}, // No scrolling in picker
	)
}

func pickerDisplayName(filename string, isScratch bool) string {
	if isScratch {
		return "[scratch]"
	}
	if filename == "" {
		return "[unnamed]"
	}
	return filepath.Base(filename)
}

// RenderOutline renders the document outline overlay centred on screen.
func (r *Renderer) RenderOutline(outline *Outline, vp *Viewport) string {
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
		return ""
	}

	// Build items for overlay.
	items := make([]OverlayItem, len(visibleItems))
	for i, item := range visibleItems {
		// Indent based on heading level.
		indent := strings.Repeat(" ", (item.Level-1)*2)
		displayText := indent + item.Text
		items[i] = OverlayItem{
			DisplayText: displayText,
			RawText:     displayText, // No ANSI codes in outline
		}
	}

	// Determine which item is selected relative to visible items.
	selectedIdx := outline.Selected - outline.ScrollOffset

	return r.RenderOverlay(
		"Document Outline",
		"Space-h",
		items,
		selectedIdx,
		vp,
		OverlayScrollInfo{
			ShowUp:   outline.ScrollOffset > 0,
			ShowDown: outline.ScrollOffset+len(visibleItems) < len(outline.Items),
		},
	)
}

// RenderBrowser renders the directory browser overlay centred on screen.
func (r *Renderer) RenderBrowser(browser *Browser, vp *Viewport) string {
	// Max visible items (use ~20 or calculate from viewport).
	maxVisible := 20
	if vp.Height-6 < maxVisible {
		maxVisible = vp.Height - 6
	}
	if maxVisible < 3 {
		maxVisible = 3
	}

	visibleItems := browser.VisibleItems(maxVisible)
	if len(visibleItems) == 0 {
		return ""
	}

	// Build items for overlay.
	items := make([]OverlayItem, len(visibleItems))
	for i, item := range visibleItems {
		displayName := item.Name
		// Format directories with blue colour and "/" suffix.
		if item.IsDir {
			displayName = "\x1b[1;34m" + item.Name + "/\x1b[0m"
		}
		items[i] = OverlayItem{
			DisplayText: displayName,
			RawText:     item.Name,
		}
	}

	// Determine which item is selected relative to visible items.
	selectedIdx := browser.Selected - browser.ScrollOffset

	return r.RenderOverlay(
		"Browse Files",
		"Space-O",
		items,
		selectedIdx,
		vp,
		OverlayScrollInfo{
			ShowUp:   browser.ScrollOffset > 0,
			ShowDown: browser.ScrollOffset+len(visibleItems) < len(browser.Items),
		},
	)
}

// RenderColumnAdjust renders the column width adjustment overlay centred on screen.
func (r *Renderer) RenderColumnAdjust(ca *ColumnAdjust, vp *Viewport) string {
	display := fmt.Sprintf("← %d →", ca.Width)
	items := []OverlayItem{
		{DisplayText: display, RawText: display},
	}

	return r.RenderOverlay(
		"Column Width",
		"Space--",
		items,
		0,
		vp,
		OverlayScrollInfo{},
	)
}

// OverlayItem represents a single item in an overlay list.
type OverlayItem struct {
	DisplayText string // The text to show (may contain ANSI codes)
	RawText     string // Plain text without ANSI (for width calculation)
}

// OverlayScrollInfo contains scroll indicator information.
type OverlayScrollInfo struct {
	ShowUp   bool // Show ↑ indicator
	ShowDown bool // Show ↓ indicator
}

// RenderOverlay renders a centred floating overlay with embedded title, proper ANSI handling,
// and reverse video only on selected content (not borders).
func (r *Renderer) RenderOverlay(
	title string,
	keybinding string,
	items []OverlayItem,
	selectedIdx int,
	vp *Viewport,
	scroll OverlayScrollInfo,
) string {
	var b strings.Builder

	// Hide cursor while overlay is shown.
	b.WriteString("\x1b[?25l")

	if len(items) == 0 {
		return b.String()
	}

	// Calculate box dimensions.
	maxTextLen := 0
	for _, item := range items {
		if len(item.RawText) > maxTextLen {
			maxTextLen = len(item.RawText)
		}
	}

	// Box width: "  > " (4) + text + "  " (2) padding + border (2)
	innerWidth := maxTextLen + 6
	// Minimum width: 60 characters (wider than before)
	if innerWidth < 60 {
		innerWidth = 60
	}
	// Embedded title format: "「Title <keybinding> "
	titleText := "╭" + "─" + title + " <" + keybinding + "> "
	if innerWidth < len(titleText)+2 {
		innerWidth = len(titleText) + 2
	}
	boxWidth := innerWidth + 2 // +2 for left/right borders
	boxHeight := len(items) + 2 // +2 for top/bottom borders

	// Centre the box.
	startCol := (vp.Width - boxWidth) / 2
	if startCol < 0 {
		startCol = 0
	}
	startRow := (vp.Height - boxHeight) / 2
	if startRow < 1 {
		startRow = 1
	}

	// Top border with embedded title: "「Title <keybinding> ─────╮"
	dashCount := innerWidth - visibleLen(titleText)
	if dashCount < 0 {
		dashCount = 0
	}
	topLine := titleText + strings.Repeat("─", dashCount + 1) + "╮"
	b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", startRow, startCol+1, topLine))

	// Item rows.
	for i, item := range items {
		row := startRow + 1 + i
		prefix := "    "
		if i == selectedIdx {
			prefix = "  > "
		}

		// Calculate padding using visibleLen to account for ANSI codes.
		visibleWidth := visibleLen(item.DisplayText)
		padding := innerWidth - 4 - visibleWidth - 2  // -2 for the explicit spaces before right border
		if padding < 0 {
			padding = 0
		}

		// Build line: only apply reverse video to selected content, not borders.
		if i == selectedIdx {
			// Selected: reverse video on content only.
			content := prefix + item.DisplayText + strings.Repeat(" ", padding) + "  "
			line := "│" + "\x1b[7m" + content + "\x1b[0m" + "│"
			b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", row, startCol+1, line))
		} else {
			// Not selected: normal rendering.
			line := "│" + prefix + item.DisplayText + strings.Repeat(" ", padding) + "  │"
			b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", row, startCol+1, line))
		}
	}

	// Bottom border.
	bottomLine := "╰" + strings.Repeat("─", innerWidth) + "╯"
	b.WriteString(fmt.Sprintf("\x1b[%d;%dH%s", startRow+boxHeight-1, startCol+1, bottomLine))

	// Scroll indicators (placed near right edge, inside border).
	// If indicator overlaps selected row, render in reverse video.
	if scroll.ShowUp {
		indicatorRow := startRow + 1
		indicatorCol := startCol + boxWidth - 2
		if selectedIdx == 0 {
			// Up arrow on selected row - use reverse video
			b.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[7m↑\x1b[0m", indicatorRow, indicatorCol))
		} else {
			b.WriteString(fmt.Sprintf("\x1b[%d;%dH↑", indicatorRow, indicatorCol))
		}
	}
	if scroll.ShowDown {
		indicatorRow := startRow + boxHeight - 2
		indicatorCol := startCol + boxWidth - 2
		if selectedIdx == len(items)-1 {
			// Down arrow on selected row - use reverse video
			b.WriteString(fmt.Sprintf("\x1b[%d;%dH\x1b[7m↓\x1b[0m", indicatorRow, indicatorCol))
		} else {
			b.WriteString(fmt.Sprintf("\x1b[%d;%dH↓", indicatorRow, indicatorCol))
		}
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

// applySpellHighlighting applies light red background highlighting to misspelled words.
// It inserts ANSI background codes while preserving existing foreground syntax highlighting.
func (r *Renderer) applySpellHighlighting(text string, displayLine DisplayLine, spellErrors []SpellError) string {
	// Find errors that overlap with this display line's character range
	displayEnd := displayLine.Offset + len([]rune(displayLine.Text))
	var relevantErrors []SpellError
	for _, err := range spellErrors {
		if err.Line == displayLine.BufferLine && err.StartCol < displayEnd && err.EndCol > displayLine.Offset {
			// Clamp to display line bounds and adjust to display-relative columns
			adjusted := err
			adjusted.StartCol = max(err.StartCol, displayLine.Offset) - displayLine.Offset
			adjusted.EndCol = min(err.EndCol, displayEnd) - displayLine.Offset
			relevantErrors = append(relevantErrors, adjusted)
		}
	}

	if len(relevantErrors) == 0 {
		return text
	}

	// Parse the text to track ANSI codes and real character positions
	runes := []rune(text)
	var result strings.Builder
	realCol := 0        // Real character position (excluding ANSI codes)
	i := 0              // Current index in runes
	inANSI := false     // Whether we're inside an ANSI escape sequence
	activeErrors := make(map[int]bool) // Track which errors are currently highlighted

	for i < len(runes) {
		r := runes[i]

		// Detect ANSI escape sequence start
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			inANSI = true
			result.WriteRune(r)
			i++
			continue
		}

		// If in ANSI sequence, copy until 'm' (end of sequence)
		if inANSI {
			result.WriteRune(r)
			if r == 'm' {
				inANSI = false
			}
			i++
			continue
		}

		// Check if we're at the start of any error
		for idx, err := range relevantErrors {
			if realCol == err.StartCol && !activeErrors[idx] {
				// Start spell error highlighting: dark text on light red background
				// Set foreground to black and background to light red
				result.WriteString("\x1b[38;5;0m\x1b[48;5;224m")
				activeErrors[idx] = true
			}
		}

		// Write the actual character
		result.WriteRune(r)
		realCol++
		i++

		// Check if we're at the end of any error
		for idx, err := range relevantErrors {
			if realCol == err.EndCol && activeErrors[idx] {
				// Reset foreground and background to restore syntax highlighting
				result.WriteString("\x1b[39m\x1b[49m")
				delete(activeErrors, idx)
			}
		}
	}

	// Close any still-active error highlighting at end of line
	if len(activeErrors) > 0 {
		result.WriteString("\x1b[39m\x1b[49m")
	}

	return result.String()
}

// applySearchHighlighting applies highlighting to search matches in the text.
// Current match gets bright yellow background, other matches get lighter yellow.
func (r *Renderer) applySearchHighlighting(text string, displayLine DisplayLine, searchActive bool, searchMatches []SearchMatch, searchCurrentIdx int) string {
	if !searchActive || len(searchMatches) == 0 {
		return text
	}

	// Find matches that overlap with this display line's character range
	displayEnd := displayLine.Offset + len([]rune(displayLine.Text))
	var relevantMatches []struct {
		match     SearchMatch
		isCurrent bool
	}
	for i, match := range searchMatches {
		if match.Line == displayLine.BufferLine && match.StartCol < displayEnd && match.EndCol > displayLine.Offset {
			// Clamp to display line bounds and adjust to display-relative columns
			adjusted := match
			adjusted.StartCol = max(match.StartCol, displayLine.Offset) - displayLine.Offset
			adjusted.EndCol = min(match.EndCol, displayEnd) - displayLine.Offset
			relevantMatches = append(relevantMatches, struct {
				match     SearchMatch
				isCurrent bool
			}{
				match:     adjusted,
				isCurrent: i == searchCurrentIdx,
			})
		}
	}

	if len(relevantMatches) == 0 {
		return text
	}

	// Parse the text to track ANSI codes and real character positions
	runes := []rune(text)
	var result strings.Builder
	realCol := 0                         // Real character position (excluding ANSI codes)
	i := 0                               // Current index in runes
	inANSI := false                      // Whether we're inside an ANSI escape sequence
	activeMatches := make(map[int]bool) // Track which matches are currently highlighted

	for i < len(runes) {
		r := runes[i]

		// Detect ANSI escape sequence start
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			inANSI = true
			result.WriteRune(r)
			i++
			continue
		}

		// If in ANSI sequence, copy until 'm' (end of sequence)
		if inANSI {
			result.WriteRune(r)
			if r == 'm' {
				inANSI = false
			}
			i++
			continue
		}

		// Check if we're at the start of any match
		for idx, rm := range relevantMatches {
			if realCol == rm.match.StartCol && !activeMatches[idx] {
				// Start search match highlighting
				if rm.isCurrent {
					// Current match: bright yellow background, black text
					result.WriteString("\x1b[38;5;0m\x1b[48;5;226m")
				} else {
					// Other matches: lighter yellow background, black text
					result.WriteString("\x1b[38;5;0m\x1b[48;5;229m")
				}
				activeMatches[idx] = true
			}
		}

		// Write the actual character
		result.WriteRune(r)
		realCol++
		i++

		// Check if we're at the end of any match
		for idx, rm := range relevantMatches {
			if realCol == rm.match.EndCol && activeMatches[idx] {
				// Reset foreground and background to restore highlighting
				result.WriteString("\x1b[39m\x1b[49m")
				delete(activeMatches, idx)
			}
		}
	}

	// Close any still-active match highlighting at end of line
	if len(activeMatches) > 0 {
		result.WriteString("\x1b[39m\x1b[49m")
	}

	return result.String()
}
