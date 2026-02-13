package main

// OpType describes the kind of edit operation for undo.
type OpType int

const (
	OpInsertChar      OpType = iota // Inserted a character
	OpDeleteChar                    // Deleted a character
	OpInsertLine                    // Inserted a newline (split line)
	OpDeleteLine                    // Deleted a newline (joined lines)
	OpInsertChars                   // Coalesced group of character inserts
	OpDeleteWholeLine               // Deleted an entire line (dd)
)

// UndoOp represents a single undoable operation or a coalesced group.
type UndoOp struct {
	Type OpType
	Line int
	Col  int
	Char rune   // For single char ops.
	Text string // For coalesced inserts.
	// Cursor position to restore after undo.
	CursorLine int
	CursorCol  int
}

// UndoStack manages the undo history with coalescing of consecutive inserts.
type UndoStack struct {
	ops      []UndoOp
	coalesce *coalesceState
}

type coalesceState struct {
	startLine int
	startCol  int
	line      int
	nextCol   int
	chars     []rune
}

func NewUndoStack() *UndoStack {
	return &UndoStack{}
}

// PushInsertChar records a character insertion, coalescing with the previous
// insert if it's at an adjacent position on the same line.
func (u *UndoStack) PushInsertChar(line, col int, ch rune) {
	if u.coalesce != nil {
		c := u.coalesce
		if line == c.line && col == c.nextCol {
			c.chars = append(c.chars, ch)
			c.nextCol = col + 1
			return
		}
		// Position changed — flush existing group.
		u.flushCoalesce()
	}
	u.coalesce = &coalesceState{
		startLine: line,
		startCol:  col,
		line:      line,
		nextCol:   col + 1,
		chars:     []rune{ch},
	}
}

// PushDeleteChar records a character deletion.
func (u *UndoStack) PushDeleteChar(line, col int, ch rune, cursorLine, cursorCol int) {
	u.flushCoalesce()
	u.ops = append(u.ops, UndoOp{
		Type:       OpDeleteChar,
		Line:       line,
		Col:        col,
		Char:       ch,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
}

// PushInsertLine records a newline insertion (line split).
func (u *UndoStack) PushInsertLine(line, col int, cursorLine, cursorCol int) {
	u.flushCoalesce()
	u.ops = append(u.ops, UndoOp{
		Type:       OpInsertLine,
		Line:       line,
		Col:        col,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
}

// PushDeleteLine records a newline deletion (line join).
func (u *UndoStack) PushDeleteLine(line, col int, cursorLine, cursorCol int) {
	u.flushCoalesce()
	u.ops = append(u.ops, UndoOp{
		Type:       OpDeleteLine,
		Line:       line,
		Col:        col,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
}

// PushDeleteWholeLine records a whole line deletion (dd operation).
func (u *UndoStack) PushDeleteWholeLine(line int, content string, cursorLine, cursorCol int) {
	u.flushCoalesce()
	u.ops = append(u.ops, UndoOp{
		Type:       OpDeleteWholeLine,
		Line:       line,
		Text:       content,
		CursorLine: cursorLine,
		CursorCol:  cursorCol,
	})
}

// flushCoalesce converts the current coalescing state into an UndoOp.
func (u *UndoStack) flushCoalesce() {
	if u.coalesce == nil {
		return
	}
	c := u.coalesce
	if len(c.chars) == 1 {
		u.ops = append(u.ops, UndoOp{
			Type:       OpInsertChar,
			Line:       c.startLine,
			Col:        c.startCol,
			Char:       c.chars[0],
			CursorLine: c.startLine,
			CursorCol:  c.startCol,
		})
	} else {
		u.ops = append(u.ops, UndoOp{
			Type:       OpInsertChars,
			Line:       c.startLine,
			Col:        c.startCol,
			Text:       string(c.chars),
			CursorLine: c.startLine,
			CursorCol:  c.startCol,
		})
	}
	u.coalesce = nil
}

// Undo pops the last operation and applies its inverse to the buffer.
// Returns the cursor position to restore, and whether an undo occurred.
func (u *UndoStack) Undo(buf *Buffer) (line, col int, ok bool) {
	u.flushCoalesce()
	if len(u.ops) == 0 {
		return 0, 0, false
	}
	op := u.ops[len(u.ops)-1]
	u.ops = u.ops[:len(u.ops)-1]

	switch op.Type {
	case OpInsertChar:
		// Undo insert: delete the character.
		runes := []rune(buf.Lines[op.Line])
		if op.Col < len(runes) {
			buf.Lines[op.Line] = string(append(runes[:op.Col], runes[op.Col+1:]...))
		}
		buf.Dirty = true
		return op.CursorLine, op.CursorCol, true

	case OpInsertChars:
		// Undo coalesced inserts: delete the range.
		runes := []rune(buf.Lines[op.Line])
		end := op.Col + len([]rune(op.Text))
		if end > len(runes) {
			end = len(runes)
		}
		buf.Lines[op.Line] = string(append(runes[:op.Col], runes[end:]...))
		buf.Dirty = true
		return op.CursorLine, op.CursorCol, true

	case OpDeleteChar:
		// Undo delete: re-insert the character.
		buf.InsertChar(op.Line, op.Col, op.Char)
		return op.CursorLine, op.CursorCol, true

	case OpInsertLine:
		// Undo newline insert: join the lines back.
		buf.JoinLines(op.Line)
		return op.CursorLine, op.CursorCol, true

	case OpDeleteLine:
		// Undo newline delete: split the line again.
		buf.InsertNewline(op.Line, op.Col)
		return op.CursorLine, op.CursorCol, true

	case OpDeleteWholeLine:
		// Undo whole line delete: re-insert the line.
		// Special case: if buffer has one empty line, replace it.
		if len(buf.Lines) == 1 && buf.Lines[0] == "" {
			buf.Lines[0] = op.Text
			buf.Dirty = true
		} else {
			buf.InsertLine(op.Line, op.Text)
		}
		return op.CursorLine, op.CursorCol, true
	}

	return 0, 0, false
}

// Len returns the number of pending undo operations.
func (u *UndoStack) Len() int {
	n := len(u.ops)
	if u.coalesce != nil {
		n++
	}
	return n
}
