package main

import "testing"

func TestUndoInsertChar(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}
	undo := NewUndoStack()

	buf.InsertChar(0, 5, '!')
	undo.PushInsertChar(0, 5, '!')

	// Force flush by pushing a different op type.
	undo.flushCoalesce()

	line, col, ok := undo.Undo(buf)
	if !ok {
		t.Fatal("undo should succeed")
	}
	if buf.Lines[0] != "hello" {
		t.Errorf("after undo: %q", buf.Lines[0])
	}
	if line != 0 || col != 5 {
		t.Errorf("cursor after undo: (%d, %d)", line, col)
	}
}

func TestUndoDeleteChar(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}
	undo := NewUndoStack()

	ch, _ := buf.DeleteChar(0, 5)
	undo.PushDeleteChar(0, 4, ch, 0, 5)

	line, col, ok := undo.Undo(buf)
	if !ok {
		t.Fatal("undo should succeed")
	}
	if buf.Lines[0] != "hello" {
		t.Errorf("after undo: %q", buf.Lines[0])
	}
	if line != 0 || col != 5 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestUndoInsertNewline(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"helloworld"}
	undo := NewUndoStack()

	undo.PushInsertLine(0, 5, 0, 5)
	buf.InsertNewline(0, 5)

	if len(buf.Lines) != 2 {
		t.Fatalf("expected 2 lines after newline, got %d", len(buf.Lines))
	}

	line, col, ok := undo.Undo(buf)
	if !ok {
		t.Fatal("undo should succeed")
	}
	if len(buf.Lines) != 1 || buf.Lines[0] != "helloworld" {
		t.Errorf("after undo: %v", buf.Lines)
	}
	if line != 0 || col != 5 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestUndoCoalescing(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{""}
	undo := NewUndoStack()

	// Simulate typing "hello" at consecutive positions.
	for i, ch := range "hello" {
		buf.InsertChar(0, i, ch)
		undo.PushInsertChar(0, i, ch)
	}

	// Should be coalesced into a single undo operation.
	undo.flushCoalesce()
	if undo.Len() != 1 {
		t.Fatalf("expected 1 coalesced op, got %d", undo.Len())
	}

	line, col, ok := undo.Undo(buf)
	if !ok {
		t.Fatal("undo should succeed")
	}
	if buf.Lines[0] != "" {
		t.Errorf("after undo coalesced insert: %q", buf.Lines[0])
	}
	if line != 0 || col != 0 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestUndoCoalescingBreaksOnGap(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{""}
	undo := NewUndoStack()

	// Type "ab" then move cursor and type "c" elsewhere.
	buf.InsertChar(0, 0, 'a')
	undo.PushInsertChar(0, 0, 'a')
	buf.InsertChar(0, 1, 'b')
	undo.PushInsertChar(0, 1, 'b')

	// Gap: insert at col 5 (non-adjacent).
	buf.Lines[0] = "ab   c"
	undo.PushInsertChar(0, 5, 'c')

	undo.flushCoalesce()
	if undo.Len() != 2 {
		t.Errorf("expected 2 ops (coalesced ab + separate c), got %d", undo.Len())
	}
}

func TestUndoEmpty(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}
	undo := NewUndoStack()

	_, _, ok := undo.Undo(buf)
	if ok {
		t.Error("undo on empty stack should return false")
	}
}

func TestUndoDeleteLine(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"helloworld"}
	undo := NewUndoStack()

	// Simulate joining: had two lines, joined at col 5.
	buf.Lines = []string{"hello", "world"}
	prevLen := buf.LineLen(0)
	buf.JoinLines(0)
	undo.PushDeleteLine(0, prevLen, 1, 0)

	if buf.Lines[0] != "helloworld" {
		t.Fatalf("after join: %q", buf.Lines[0])
	}

	line, col, ok := undo.Undo(buf)
	if !ok {
		t.Fatal("undo should succeed")
	}
	if len(buf.Lines) != 2 || buf.Lines[0] != "hello" || buf.Lines[1] != "world" {
		t.Errorf("after undo join: %v", buf.Lines)
	}
	if line != 1 || col != 0 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestUndoDeleteWholeLine(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"first", "second", "third"}
	undo := NewUndoStack()

	// Delete the middle line (dd operation).
	content := buf.DeleteLine(1)
	undo.PushDeleteWholeLine(1, content, 1, 0)

	if len(buf.Lines) != 2 || buf.Lines[0] != "first" || buf.Lines[1] != "third" {
		t.Fatalf("after delete whole line: %v", buf.Lines)
	}

	line, col, ok := undo.Undo(buf)
	if !ok {
		t.Fatal("undo should succeed")
	}
	if len(buf.Lines) != 3 || buf.Lines[0] != "first" || buf.Lines[1] != "second" || buf.Lines[2] != "third" {
		t.Errorf("after undo delete whole line: %v", buf.Lines)
	}
	if line != 1 || col != 0 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestUndoDeleteWholeLineSingle(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"only line"}
	undo := NewUndoStack()

	// Delete the only line (should clear it).
	content := buf.DeleteLine(0)
	undo.PushDeleteWholeLine(0, content, 0, 0)

	if len(buf.Lines) != 1 || buf.Lines[0] != "" {
		t.Fatalf("after delete single line: %v", buf.Lines)
	}

	line, col, ok := undo.Undo(buf)
	if !ok {
		t.Fatal("undo should succeed")
	}
	if len(buf.Lines) != 1 || buf.Lines[0] != "only line" {
		t.Errorf("after undo single line delete: %v", buf.Lines)
	}
	if line != 0 || col != 0 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

// --- v1.5.0 redo tests ---

func TestRedoInsertChar(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}
	undo := NewUndoStack()

	buf.InsertChar(0, 5, '!')
	undo.PushInsertChar(0, 5, '!')
	undo.flushCoalesce()

	// Undo the insert.
	undo.Undo(buf)
	if buf.Lines[0] != "hello" {
		t.Errorf("after undo: %q", buf.Lines[0])
	}

	// Redo the insert.
	line, col, ok := undo.Redo(buf)
	if !ok {
		t.Fatal("redo should succeed")
	}
	if buf.Lines[0] != "hello!" {
		t.Errorf("after redo: %q", buf.Lines[0])
	}
	if line != 0 || col != 6 {
		t.Errorf("cursor after redo: (%d, %d)", line, col)
	}
}

func TestRedoDeleteChar(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}
	undo := NewUndoStack()

	ch, _ := buf.DeleteChar(0, 5)
	undo.PushDeleteChar(0, 4, ch, 0, 5)

	// Undo the delete.
	undo.Undo(buf)
	if buf.Lines[0] != "hello" {
		t.Errorf("after undo: %q", buf.Lines[0])
	}

	// Redo the delete.
	line, col, ok := undo.Redo(buf)
	if !ok {
		t.Fatal("redo should succeed")
	}
	if buf.Lines[0] != "hell" {
		t.Errorf("after redo: %q", buf.Lines[0])
	}
	if line != 0 || col != 5 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestRedoInsertLine(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"helloworld"}
	undo := NewUndoStack()

	undo.PushInsertLine(0, 5, 0, 5)
	buf.InsertNewline(0, 5)

	// Undo the newline insert.
	undo.Undo(buf)
	if len(buf.Lines) != 1 || buf.Lines[0] != "helloworld" {
		t.Errorf("after undo: %v", buf.Lines)
	}

	// Redo the newline insert.
	line, col, ok := undo.Redo(buf)
	if !ok {
		t.Fatal("redo should succeed")
	}
	if len(buf.Lines) != 2 || buf.Lines[0] != "hello" || buf.Lines[1] != "world" {
		t.Errorf("after redo: %v", buf.Lines)
	}
	if line != 1 || col != 0 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestRedoDeleteWholeLine(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"first", "second", "third"}
	undo := NewUndoStack()

	content := buf.DeleteLine(1)
	undo.PushDeleteWholeLine(1, content, 1, 0)

	// Undo the delete.
	undo.Undo(buf)
	if len(buf.Lines) != 3 || buf.Lines[1] != "second" {
		t.Errorf("after undo: %v", buf.Lines)
	}

	// Redo the delete.
	line, col, ok := undo.Redo(buf)
	if !ok {
		t.Fatal("redo should succeed")
	}
	if len(buf.Lines) != 2 || buf.Lines[0] != "first" || buf.Lines[1] != "third" {
		t.Errorf("after redo: %v", buf.Lines)
	}
	if line != 1 || col != 0 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestRedoInsertWholeLine(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"first", "second"}
	undo := NewUndoStack()

	buf.InsertLine(1, "")
	undo.PushInsertWholeLine(1)

	// Undo the insert.
	undo.Undo(buf)
	if len(buf.Lines) != 2 || buf.Lines[0] != "first" || buf.Lines[1] != "second" {
		t.Errorf("after undo: %v", buf.Lines)
	}

	// Redo the insert.
	line, col, ok := undo.Redo(buf)
	if !ok {
		t.Fatal("redo should succeed")
	}
	if len(buf.Lines) != 3 || buf.Lines[1] != "" {
		t.Errorf("after redo: %v", buf.Lines)
	}
	if line != 1 || col != 0 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestRedoCoalescedInserts(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{""}
	undo := NewUndoStack()

	// Type "hello" at consecutive positions.
	for i, ch := range "hello" {
		buf.InsertChar(0, i, ch)
		undo.PushInsertChar(0, i, ch)
	}
	undo.flushCoalesce()

	// Undo the coalesced insert.
	undo.Undo(buf)
	if buf.Lines[0] != "" {
		t.Errorf("after undo: %q", buf.Lines[0])
	}

	// Redo the coalesced insert.
	line, col, ok := undo.Redo(buf)
	if !ok {
		t.Fatal("redo should succeed")
	}
	if buf.Lines[0] != "hello" {
		t.Errorf("after redo: %q", buf.Lines[0])
	}
	if line != 0 || col != 5 {
		t.Errorf("cursor: (%d, %d)", line, col)
	}
}

func TestRedoEmpty(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}
	undo := NewUndoStack()

	_, _, ok := undo.Redo(buf)
	if ok {
		t.Error("redo on empty redo stack should return false")
	}
}

func TestRedoStackClearedOnNewOp(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}
	undo := NewUndoStack()

	// Insert a character.
	buf.InsertChar(0, 5, '!')
	undo.PushInsertChar(0, 5, '!')
	undo.flushCoalesce()

	// Undo it.
	undo.Undo(buf)

	// Insert a new character (should clear redo stack).
	buf.InsertChar(0, 5, '?')
	undo.PushInsertChar(0, 5, '?')

	// Try to redo — should fail because redo stack was cleared.
	_, _, ok := undo.Redo(buf)
	if ok {
		t.Error("redo should fail after new operation")
	}
}

func TestMultipleUndoRedo(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{""}
	undo := NewUndoStack()

	// Insert 'a'.
	buf.InsertChar(0, 0, 'a')
	undo.PushInsertChar(0, 0, 'a')
	undo.flushCoalesce()

	// Insert 'b'.
	buf.InsertChar(0, 1, 'b')
	undo.PushInsertChar(0, 1, 'b')
	undo.flushCoalesce()

	// Now: "ab"
	if buf.Lines[0] != "ab" {
		t.Fatalf("after inserts: %q", buf.Lines[0])
	}

	// Undo twice.
	undo.Undo(buf) // Remove 'b'
	if buf.Lines[0] != "a" {
		t.Errorf("after first undo: %q", buf.Lines[0])
	}
	undo.Undo(buf) // Remove 'a'
	if buf.Lines[0] != "" {
		t.Errorf("after second undo: %q", buf.Lines[0])
	}

	// Redo twice.
	undo.Redo(buf) // Restore 'a'
	if buf.Lines[0] != "a" {
		t.Errorf("after first redo: %q", buf.Lines[0])
	}
	undo.Redo(buf) // Restore 'b'
	if buf.Lines[0] != "ab" {
		t.Errorf("after second redo: %q", buf.Lines[0])
	}
}
