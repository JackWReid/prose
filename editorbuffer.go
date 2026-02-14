package main

import (
	"path/filepath"
	"strings"
	"time"
)

// EditorBuffer holds all per-buffer state: text, undo history, cursor, scroll, and highlighter.
type EditorBuffer struct {
	buf          *Buffer
	undo         *UndoStack
	highlighter  Highlighter
	cursorLine   int
	cursorCol    int
	scrollOffset int
	isScratch    bool // True if this is the session scratch buffer

	// Spell checking state
	spellErrors       []SpellError  // Cached spell errors
	spellCheckPending bool          // Debounce flag
	lastEdit          time.Time     // Last edit timestamp

	// Search state
	searchActive     bool
	searchQuery      string
	searchMatches    []SearchMatch
	searchCurrentIdx int // -1 when no current match
}

// SearchMatch represents a single search match in the buffer.
type SearchMatch struct {
	Line     int // Buffer line number
	StartCol int // Starting rune index
	EndCol   int // Ending rune index
}

// NewEditorBuffer creates a new EditorBuffer for the given filename.
func NewEditorBuffer(filename string) *EditorBuffer {
	return &EditorBuffer{
		buf:         NewBuffer(filename),
		undo:        NewUndoStack(),
		highlighter: DetectHighlighter(filename),
	}
}

// Filename returns the buffer's filename.
func (eb *EditorBuffer) Filename() string {
	return eb.buf.Filename
}

// IsDirty returns whether the buffer has unsaved changes.
// Scratch buffers are never considered dirty (they're not saved).
func (eb *EditorBuffer) IsDirty() bool {
	if eb.isScratch {
		return false
	}
	return eb.buf.Dirty
}

// WordCount returns the word count of the buffer.
func (eb *EditorBuffer) WordCount() int {
	return eb.buf.WordCount()
}

// ShouldSpellCheck returns whether spell checking should be enabled for this buffer.
// Only .md and .txt files are spell checked.
func (eb *EditorBuffer) ShouldSpellCheck() bool {
	if eb.buf.Filename == "" {
		return false
	}

	ext := strings.ToLower(filepath.Ext(eb.buf.Filename))
	return ext == ".md" || ext == ".txt" || ext == ".markdown"
}

// SpellErrorCount returns the number of cached spell errors.
func (eb *EditorBuffer) SpellErrorCount() int {
	return len(eb.spellErrors)
}

// ScheduleSpellCheck marks that a spell check should be performed after debouncing.
func (eb *EditorBuffer) ScheduleSpellCheck() {
	if !eb.ShouldSpellCheck() {
		return
	}
	eb.spellCheckPending = true
	eb.lastEdit = time.Now()
}

// PerformSpellCheck runs spell checking if enough time has elapsed since the last edit.
// This implements debouncing to avoid checking on every keystroke.
func (eb *EditorBuffer) PerformSpellCheck(spellChecker *SpellChecker) {
	if !eb.spellCheckPending {
		return
	}

	// Debounce: only check if 300ms have elapsed since last edit
	elapsed := time.Since(eb.lastEdit)
	if elapsed < 300*time.Millisecond {
		return
	}

	// Clear pending flag
	eb.spellCheckPending = false

	// Clear previous errors
	eb.spellErrors = nil

	// Check all lines for spelling errors
	for i := 0; i < len(eb.buf.Lines); i++ {
		lineErrors := spellChecker.CheckLine(i, eb.buf.Lines[i])
		eb.spellErrors = append(eb.spellErrors, lineErrors...)
	}
}
