package main

// EditorBuffer holds all per-buffer state: text, undo history, cursor, scroll, and highlighter.
type EditorBuffer struct {
	buf          *Buffer
	undo         *UndoStack
	highlighter  Highlighter
	cursorLine   int
	cursorCol    int
	scrollOffset int
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
func (eb *EditorBuffer) IsDirty() bool {
	return eb.buf.Dirty
}

// WordCount returns the word count of the buffer.
func (eb *EditorBuffer) WordCount() int {
	return eb.buf.WordCount()
}
