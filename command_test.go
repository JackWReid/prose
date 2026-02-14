package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestApp creates a minimal App for testing executeCommand.
func newTestApp(filename string) *App {
	eb := NewEditorBuffer(filename)
	return &App{
		buffers:   []*EditorBuffer{eb},
		renderer:  NewRenderer(),
		statusBar: NewStatusBar(),
		picker:    &Picker{},
		mode:      ModeDefault,
	}
}

func TestCommandQuit(t *testing.T) {
	a := newTestApp("test.txt")
	a.executeCommand("q")
	if !a.quit {
		t.Error(":q on clean buffer should quit")
	}
}

func TestCommandQuitDirty(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Dirty = true
	a.executeCommand("q")
	if a.quit {
		t.Error(":q on dirty buffer should not quit")
	}
	if a.statusBar.StatusMessage == "" {
		t.Error(":q on dirty buffer should show error message")
	}
}

func TestCommandForceQuit(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Dirty = true
	a.executeCommand("q!")
	if !a.quit {
		t.Error(":q! should force quit even on dirty buffer")
	}
}

func TestCommandWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	a := newTestApp(path)
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().buf.Dirty = true

	a.executeCommand("w")

	if a.currentBuf().buf.Dirty {
		t.Error(":w should clear dirty flag")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "hello\n" {
		t.Errorf("saved content: %q", string(data))
	}
}

func TestCommandWriteUnnamed(t *testing.T) {
	a := newTestApp("")
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().buf.Dirty = true

	a.executeCommand("w")

	// Should start a PromptSaveNew since buffer has no filename.
	if a.statusBar.Prompt != PromptSaveNew {
		t.Errorf("expected PromptSaveNew, got %v", a.statusBar.Prompt)
	}
}

func TestCommandWriteFilename(t *testing.T) {
	dir := t.TempDir()
	origPath := filepath.Join(dir, "original.txt")
	newPath := filepath.Join(dir, "newfile.txt")

	a := newTestApp(origPath)
	a.currentBuf().buf.Lines = []string{"content"}
	a.currentBuf().buf.Dirty = true

	a.executeCommand("w " + newPath)

	if a.currentBuf().buf.Filename != newPath {
		t.Errorf("filename should be updated to %q, got %q", newPath, a.currentBuf().buf.Filename)
	}
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(data) != "content\n" {
		t.Errorf("saved content: %q", string(data))
	}
}

func TestCommandWriteQuit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	a := newTestApp(path)
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().buf.Dirty = true

	a.executeCommand("wq")

	if !a.quit {
		t.Error(":wq should quit")
	}
	if a.currentBuf().buf.Dirty {
		t.Error(":wq should save")
	}
}

func TestCommandWriteQuitUnnamed(t *testing.T) {
	a := newTestApp("")
	a.currentBuf().buf.Lines = []string{"hello"}

	a.executeCommand("wq")

	if a.quit {
		t.Error(":wq on unnamed should not quit immediately")
	}
	if !a.quitAfterSave {
		t.Error(":wq on unnamed should set quitAfterSave")
	}
	if a.statusBar.Prompt != PromptSaveNew {
		t.Errorf("expected PromptSaveNew, got %v", a.statusBar.Prompt)
	}
}

func TestCommandRename(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	os.WriteFile(oldPath, []byte("data\n"), 0644)

	a := newTestApp(oldPath)
	a.currentBuf().buf.Lines = []string{"data"}

	a.executeCommand("rename " + newPath)

	if a.currentBuf().buf.Filename != newPath {
		t.Errorf("filename should be %q, got %q", newPath, a.currentBuf().buf.Filename)
	}
	// Old file should no longer exist.
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old file should be removed after rename")
	}
	// New file should exist.
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read renamed file: %v", err)
	}
	if string(data) != "data\n" {
		t.Errorf("renamed file content: %q", string(data))
	}
}

func TestCommandRenameUnnamed(t *testing.T) {
	dir := t.TempDir()
	newPath := filepath.Join(dir, "newfile.txt")

	a := newTestApp("")
	a.currentBuf().buf.Lines = []string{"content"}
	a.currentBuf().buf.Dirty = true

	a.executeCommand("rename " + newPath)

	// Should behave like :w <filename> for unnamed buffers.
	if a.currentBuf().buf.Filename != newPath {
		t.Errorf("filename should be %q, got %q", newPath, a.currentBuf().buf.Filename)
	}
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != "content\n" {
		t.Errorf("file content: %q", string(data))
	}
}

func TestCommandWriteQuitUnnamedFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "saved.txt")

	a := newTestApp("")
	a.currentBuf().buf.Lines = []string{"hello"}

	// Simulate :wq on unnamed buffer.
	a.executeCommand("wq")

	// Now simulate typing the filename and pressing Enter in the prompt.
	for _, ch := range path {
		a.handlePromptKey(Key{Type: KeyRune, Rune: ch})
	}
	a.handlePromptKey(Key{Type: KeyEnter})

	if !a.quit {
		t.Error("should quit after save-as completes")
	}
	if a.currentBuf().buf.Filename != path {
		t.Errorf("filename should be %q, got %q", path, a.currentBuf().buf.Filename)
	}
}

func TestCommandWriteQuitUnnamedCancel(t *testing.T) {
	a := newTestApp("")
	a.currentBuf().buf.Lines = []string{"hello"}

	a.executeCommand("wq")
	// Cancel the prompt.
	a.handlePromptKey(Key{Type: KeyEscape})

	if a.quit {
		t.Error("should not quit after cancelling save-as")
	}
	if a.quitAfterSave {
		t.Error("quitAfterSave should be reset on cancel")
	}
}

func TestCommandUnknown(t *testing.T) {
	a := newTestApp("test.txt")
	a.executeCommand("foobar")
	if a.statusBar.StatusMessage != "Unknown command: foobar" {
		t.Errorf("unknown command message: %q", a.statusBar.StatusMessage)
	}
}

// --- Multi-buffer command tests ---

func TestCommandEditOpensNewBuffer(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")
	os.WriteFile(file1, []byte("one\n"), 0644)
	os.WriteFile(file2, []byte("two\n"), 0644)

	a := newTestApp(file1)
	a.currentBuf().buf.Load()

	a.executeCommand("e " + file2)

	if len(a.buffers) != 2 {
		t.Fatalf("expected 2 buffers, got %d", len(a.buffers))
	}
	if a.currentBuffer != 1 {
		t.Errorf("currentBuffer = %d, want 1", a.currentBuffer)
	}
	if a.currentBuf().Filename() != file2 {
		t.Errorf("active filename = %q, want %q", a.currentBuf().Filename(), file2)
	}
}

func TestCommandEditSwitchesToExisting(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "file1.txt")
	os.WriteFile(file1, []byte("one\n"), 0644)

	a := newTestApp(file1)
	a.currentBuf().buf.Load()

	// Open same file again — should not duplicate.
	a.executeCommand("e " + file1)

	if len(a.buffers) != 1 {
		t.Fatalf("expected 1 buffer (no duplicate), got %d", len(a.buffers))
	}
	if a.currentBuffer != 0 {
		t.Errorf("currentBuffer = %d, want 0", a.currentBuffer)
	}
}

func TestCommandEditNoArgs(t *testing.T) {
	a := newTestApp("test.txt")
	a.executeCommand("e")
	if a.statusBar.StatusMessage != "Usage: :e <filename>" {
		t.Errorf("expected usage message, got %q", a.statusBar.StatusMessage)
	}
}

func TestCommandQuitClosesBuffer(t *testing.T) {
	a := newTestApp("file1.txt")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))
	a.buffers = append(a.buffers, NewEditorBuffer("file3.txt"))

	// Close the first buffer.
	a.executeCommand("q")

	if a.quit {
		t.Error("should not quit with remaining buffers")
	}
	if len(a.buffers) != 2 {
		t.Fatalf("expected 2 buffers, got %d", len(a.buffers))
	}
}

func TestCommandQuitLastBuffer(t *testing.T) {
	a := newTestApp("file1.txt")
	a.executeCommand("q")
	if !a.quit {
		t.Error("should quit when closing last buffer")
	}
}

func TestCommandForceQuitClosesBuffer(t *testing.T) {
	a := newTestApp("file1.txt")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))
	a.currentBuf().buf.Dirty = true

	a.executeCommand("q!")

	if a.quit {
		t.Error("should not quit with remaining buffers")
	}
	if len(a.buffers) != 1 {
		t.Fatalf("expected 1 buffer, got %d", len(a.buffers))
	}
}

func TestCommandQuitAdjustsIndex(t *testing.T) {
	a := newTestApp("file1.txt")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))
	a.currentBuffer = 1

	a.executeCommand("q")

	if a.currentBuffer != 0 {
		t.Errorf("currentBuffer should be 0 after closing last index, got %d", a.currentBuffer)
	}
}

// --- v1.4.0 feature tests ---

func TestDeleteWholeLine(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1
	a.currentBuf().cursorCol = 3

	a.deleteWholeLine()

	if len(a.currentBuf().buf.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[0] != "first" || a.currentBuf().buf.Lines[1] != "third" {
		t.Errorf("lines after dd: %v", a.currentBuf().buf.Lines)
	}
	if a.currentBuf().cursorLine != 1 {
		t.Errorf("cursor line: %d", a.currentBuf().cursorLine)
	}
}

func TestDeleteWholeLineSingleLine(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"only line"}
	a.currentBuf().cursorLine = 0
	a.currentBuf().cursorCol = 5

	a.deleteWholeLine()

	if len(a.currentBuf().buf.Lines) != 1 || a.currentBuf().buf.Lines[0] != "" {
		t.Errorf("single line dd should clear to empty: %v", a.currentBuf().buf.Lines)
	}
	if a.currentBuf().cursorLine != 0 || a.currentBuf().cursorCol != 0 {
		t.Errorf("cursor: (%d, %d)", a.currentBuf().cursorLine, a.currentBuf().cursorCol)
	}
}

func TestDeleteWholeLineLastLine(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second"}
	a.currentBuf().cursorLine = 1

	a.deleteWholeLine()

	if len(a.currentBuf().buf.Lines) != 1 || a.currentBuf().buf.Lines[0] != "first" {
		t.Errorf("after deleting last line: %v", a.currentBuf().buf.Lines)
	}
	if a.currentBuf().cursorLine != 0 {
		t.Errorf("cursor should move to line 0, got %d", a.currentBuf().cursorLine)
	}
}

func TestHandleDDOperator(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1

	// Press 'd' once.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})
	if !a.dPending {
		t.Error("first 'd' should set dPending")
	}

	// Press 'd' again.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})
	if a.dPending {
		t.Error("second 'd' should clear dPending")
	}
	if len(a.currentBuf().buf.Lines) != 2 {
		t.Fatalf("dd should delete line, got %d lines", len(a.currentBuf().buf.Lines))
	}
}

func TestHandleDDCancellation(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second"}

	// Press 'd' then something else.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'j'})

	if a.dPending {
		t.Error("dPending should be cleared by non-d key")
	}
	if len(a.currentBuf().buf.Lines) != 2 {
		t.Error("dj should not delete anything")
	}
}

func TestMotionA(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().cursorCol = 2

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'A'})

	if a.mode != ModeEdit {
		t.Error("'A' should enter edit mode")
	}
	if a.currentBuf().cursorCol != 5 {
		t.Errorf("'A' should move to end of line, got col %d", a.currentBuf().cursorCol)
	}
}

func TestMotionCaret(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"   hello"}
	a.currentBuf().cursorCol = 7

	a.handleDefaultKey(Key{Type: KeyRune, Rune: '^'})

	if a.currentBuf().cursorCol != 3 {
		t.Errorf("'^' should jump to first non-space, got col %d", a.currentBuf().cursorCol)
	}
}

func TestMotionCaretAllSpaces(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"     "}
	a.currentBuf().cursorCol = 3

	a.handleDefaultKey(Key{Type: KeyRune, Rune: '^'})

	if a.currentBuf().cursorCol != 0 {
		t.Errorf("'^' on all-space line should go to col 0, got %d", a.currentBuf().cursorCol)
	}
}

func TestMotionDollar(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().cursorCol = 0

	a.handleDefaultKey(Key{Type: KeyRune, Rune: '$'})

	if a.currentBuf().cursorCol != 5 {
		t.Errorf("'$' should jump to end of line, got col %d", a.currentBuf().cursorCol)
	}
}

func TestScrollDown(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"1", "2", "3", "4", "5"}
	a.currentBuf().cursorLine = 0

	a.scrollDown(2)

	if a.currentBuf().cursorLine != 2 {
		t.Errorf("scrollDown(2) should move to line 2, got %d", a.currentBuf().cursorLine)
	}
}

func TestScrollDownClamped(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"1", "2", "3"}
	a.currentBuf().cursorLine = 1

	a.scrollDown(10)

	if a.currentBuf().cursorLine != 2 {
		t.Errorf("scrollDown past end should clamp, got line %d", a.currentBuf().cursorLine)
	}
}

func TestScrollUp(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"1", "2", "3", "4", "5"}
	a.currentBuf().cursorLine = 4

	a.scrollUp(2)

	if a.currentBuf().cursorLine != 2 {
		t.Errorf("scrollUp(2) should move to line 2, got %d", a.currentBuf().cursorLine)
	}
}

func TestScrollUpClamped(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"1", "2", "3"}
	a.currentBuf().cursorLine = 1

	a.scrollUp(10)

	if a.currentBuf().cursorLine != 0 {
		t.Errorf("scrollUp past start should clamp, got line %d", a.currentBuf().cursorLine)
	}
}

func TestAppDeleteCharForward(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().cursorCol = 1
	a.mode = ModeEdit

	a.deleteCharForward()

	if a.currentBuf().buf.Lines[0] != "hllo" {
		t.Errorf("forward delete should remove 'e', got %q", a.currentBuf().buf.Lines[0])
	}
	if a.currentBuf().cursorCol != 1 {
		t.Errorf("cursor should stay at col 1, got %d", a.currentBuf().cursorCol)
	}
}

func TestAppDeleteCharForwardAtEnd(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"hello", "world"}
	a.currentBuf().cursorLine = 0
	a.currentBuf().cursorCol = 5
	a.mode = ModeEdit

	a.deleteCharForward()

	if len(a.currentBuf().buf.Lines) != 1 || a.currentBuf().buf.Lines[0] != "helloworld" {
		t.Errorf("forward delete at end should join lines, got %v", a.currentBuf().buf.Lines)
	}
}

func TestHomeEndDefaultMode(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().cursorCol = 2

	a.handleDefaultKey(Key{Type: KeyHome})
	if a.currentBuf().cursorCol != 0 {
		t.Errorf("Home in default mode: got col %d", a.currentBuf().cursorCol)
	}

	a.handleDefaultKey(Key{Type: KeyEnd})
	if a.currentBuf().cursorCol != 5 {
		t.Errorf("End in default mode: got col %d", a.currentBuf().cursorCol)
	}
}

func TestHomeEndEditMode(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"hello"}
	a.currentBuf().cursorCol = 2
	a.mode = ModeEdit

	a.handleEditKey(Key{Type: KeyHome})
	if a.currentBuf().cursorCol != 0 {
		t.Errorf("Home in edit mode: got col %d", a.currentBuf().cursorCol)
	}

	a.handleEditKey(Key{Type: KeyEnd})
	if a.currentBuf().cursorCol != 5 {
		t.Errorf("End in edit mode: got col %d", a.currentBuf().cursorCol)
	}
}

// --- v1.5.0 feature tests ---

func TestOCommandInsertLineAbove(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1
	a.currentBuf().cursorCol = 3

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'O'})

	if len(a.currentBuf().buf.Lines) != 4 {
		t.Fatalf("expected 4 lines after O, got %d", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[1] != "" {
		t.Errorf("inserted line should be empty, got %q", a.currentBuf().buf.Lines[1])
	}
	if a.currentBuf().buf.Lines[2] != "second" {
		t.Errorf("original line should be pushed down, got %q", a.currentBuf().buf.Lines[2])
	}
	if a.mode != ModeEdit {
		t.Error("O should enter edit mode")
	}
	if a.currentBuf().cursorLine != 1 || a.currentBuf().cursorCol != 0 {
		t.Errorf("cursor should be at (1, 0), got (%d, %d)", a.currentBuf().cursorLine, a.currentBuf().cursorCol)
	}
}

func TestOCommandAtFirstLine(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first"}
	a.currentBuf().cursorLine = 0

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'O'})

	if len(a.currentBuf().buf.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[0] != "" {
		t.Errorf("inserted line should be empty, got %q", a.currentBuf().buf.Lines[0])
	}
	if a.currentBuf().cursorLine != 0 {
		t.Errorf("cursor should stay at line 0, got %d", a.currentBuf().cursorLine)
	}
}

func TestGGMotion(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third", "fourth"}
	a.currentBuf().cursorLine = 2
	a.currentBuf().cursorCol = 5

	// Press 'g' once.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'g'})
	if !a.gPending {
		t.Error("first 'g' should set gPending")
	}

	// Press 'g' again.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'g'})
	if a.gPending {
		t.Error("second 'g' should clear gPending")
	}
	if a.currentBuf().cursorLine != 0 {
		t.Errorf("gg should jump to line 0, got %d", a.currentBuf().cursorLine)
	}
	if a.currentBuf().cursorCol != 0 {
		t.Errorf("gg should move to col 0, got %d", a.currentBuf().cursorCol)
	}
}

func TestGGCancellation(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second"}
	a.currentBuf().cursorLine = 1

	// Press 'g' then something else.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'g'})
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'j'})

	if a.gPending {
		t.Error("gPending should be cleared by non-g key")
	}
	if a.currentBuf().cursorLine != 1 {
		t.Error("gj should not jump anywhere")
	}
}

func TestGMotion(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third", "fourth"}
	a.currentBuf().cursorLine = 1
	a.currentBuf().cursorCol = 5

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'G'})

	if a.currentBuf().cursorLine != 3 {
		t.Errorf("G should jump to last line (3), got %d", a.currentBuf().cursorLine)
	}
	if a.currentBuf().cursorCol != 0 {
		t.Errorf("G should move to col 0, got %d", a.currentBuf().cursorCol)
	}
}

func TestYYYank(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1

	// Press 'y' once.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'y'})
	if !a.yPending {
		t.Error("first 'y' should set yPending")
	}

	// Press 'y' again.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'y'})
	if a.yPending {
		t.Error("second 'y' should clear yPending")
	}
	if a.yankBuffer != "second" {
		t.Errorf("yankBuffer should be 'second', got %q", a.yankBuffer)
	}
	// Lines should be unchanged.
	if len(a.currentBuf().buf.Lines) != 3 {
		t.Error("yy should not modify lines")
	}
}

func TestYYCancellation(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second"}

	// Press 'y' then something else.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'y'})
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'j'})

	if a.yPending {
		t.Error("yPending should be cleared by non-y key")
	}
	if a.yankBuffer != "" {
		t.Error("yankBuffer should remain empty after cancelled yy")
	}
}

func TestPasteBelow(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second"}
	a.currentBuf().cursorLine = 0
	a.yankBuffer = "pasted"

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'p'})

	if len(a.currentBuf().buf.Lines) != 3 {
		t.Fatalf("expected 3 lines after paste, got %d", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[1] != "pasted" {
		t.Errorf("pasted line should be 'pasted', got %q", a.currentBuf().buf.Lines[1])
	}
	if a.currentBuf().cursorLine != 1 {
		t.Errorf("cursor should move to line 1, got %d", a.currentBuf().cursorLine)
	}
}

func TestPasteAbove(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second"}
	a.currentBuf().cursorLine = 1
	a.yankBuffer = "pasted"

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'P'})

	if len(a.currentBuf().buf.Lines) != 3 {
		t.Fatalf("expected 3 lines after paste, got %d", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[1] != "pasted" {
		t.Errorf("pasted line should be 'pasted', got %q", a.currentBuf().buf.Lines[1])
	}
	if a.currentBuf().buf.Lines[2] != "second" {
		t.Errorf("original line should be pushed down, got %q", a.currentBuf().buf.Lines[2])
	}
	if a.currentBuf().cursorLine != 1 {
		t.Errorf("cursor should stay at line 1, got %d", a.currentBuf().cursorLine)
	}
}

func TestPasteEmptyBuffer(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first"}
	a.yankBuffer = ""

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'p'})

	// Should be no-op.
	if len(a.currentBuf().buf.Lines) != 1 {
		t.Error("paste with empty buffer should be no-op")
	}
}

func TestDDPopulatesYankBuffer(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1

	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})

	if a.yankBuffer != "second" {
		t.Errorf("dd should populate yankBuffer, got %q", a.yankBuffer)
	}
	if len(a.currentBuf().buf.Lines) != 2 {
		t.Fatalf("dd should delete line, got %d lines", len(a.currentBuf().buf.Lines))
	}
}

func TestDDThenPaste(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1

	// Delete second line with dd.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})

	// Now paste it back below.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'p'})

	if len(a.currentBuf().buf.Lines) != 3 {
		t.Fatalf("expected 3 lines after dd+p, got %d", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[0] != "first" || a.currentBuf().buf.Lines[1] != "third" || a.currentBuf().buf.Lines[2] != "second" {
		t.Errorf("unexpected lines after dd+p: %v", a.currentBuf().buf.Lines)
	}
}

func TestUndoWithU(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1

	// Delete second line with dd.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})

	if len(a.currentBuf().buf.Lines) != 2 {
		t.Fatalf("dd should delete line, got %d lines", len(a.currentBuf().buf.Lines))
	}

	// Undo with 'u'.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'u'})

	if len(a.currentBuf().buf.Lines) != 3 {
		t.Fatalf("u should restore deleted line, got %d lines", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[1] != "second" {
		t.Errorf("restored line should be 'second', got %q", a.currentBuf().buf.Lines[1])
	}
}

func TestRedoWithCtrlR(t *testing.T) {
	a := newTestApp("test.txt")
	a.currentBuf().buf.Lines = []string{"first", "second", "third"}
	a.currentBuf().cursorLine = 1

	// Delete second line with dd.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'd'})

	// Undo with 'u'.
	a.handleDefaultKey(Key{Type: KeyRune, Rune: 'u'})

	if len(a.currentBuf().buf.Lines) != 3 {
		t.Fatalf("expected 3 lines after undo, got %d", len(a.currentBuf().buf.Lines))
	}

	// Redo with Ctrl+R.
	a.handleDefaultKey(Key{Type: KeyCtrlR})

	if len(a.currentBuf().buf.Lines) != 2 {
		t.Fatalf("Ctrl+R should redo delete, got %d lines", len(a.currentBuf().buf.Lines))
	}
	if a.currentBuf().buf.Lines[0] != "first" || a.currentBuf().buf.Lines[1] != "third" {
		t.Errorf("lines after redo: %v", a.currentBuf().buf.Lines)
	}
}

// --- v1.7.0 feature tests: Quit-all commands ---

func TestCommandQuitAll(t *testing.T) {
	a := newTestApp("file1.txt")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))
	a.buffers = append(a.buffers, NewEditorBuffer("file3.txt"))

	a.executeCommand("qa")

	if !a.quit {
		t.Error(":qa should quit when all buffers are clean")
	}
}

func TestCommandQuitAllWithDirty(t *testing.T) {
	a := newTestApp("file1.txt")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))
	a.buffers = append(a.buffers, NewEditorBuffer("file3.txt"))

	// Make two buffers dirty.
	a.buffers[0].buf.Dirty = true
	a.buffers[2].buf.Dirty = true

	a.executeCommand("qa")

	if a.quit {
		t.Error(":qa should not quit when buffers have unsaved changes")
	}
	if a.statusBar.StatusMessage == "" {
		t.Error(":qa with dirty buffers should show error message")
	}
	if !strings.Contains(a.statusBar.StatusMessage, "2 buffer(s)") {
		t.Errorf("error message should mention 2 dirty buffers, got: %q", a.statusBar.StatusMessage)
	}
}

func TestCommandForceQuitAll(t *testing.T) {
	a := newTestApp("file1.txt")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))
	a.buffers[0].buf.Dirty = true
	a.buffers[1].buf.Dirty = true

	a.executeCommand("qa!")

	if !a.quit {
		t.Error(":qa! should force quit even with dirty buffers")
	}
}

func TestCommandForceQuitAllAlternate(t *testing.T) {
	a := newTestApp("file1.txt")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))
	a.buffers[0].buf.Dirty = true

	a.executeCommand("!qa")

	if !a.quit {
		t.Error(":!qa should force quit even with dirty buffers")
	}
}

func TestCommandWriteQuitAll(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "file1.txt")
	path2 := filepath.Join(dir, "file2.txt")

	a := newTestApp(path1)
	a.buffers = append(a.buffers, NewEditorBuffer(path2))

	// Make both dirty.
	a.buffers[0].buf.Lines = []string{"content1"}
	a.buffers[0].buf.Dirty = true
	a.buffers[1].buf.Lines = []string{"content2"}
	a.buffers[1].buf.Dirty = true

	a.executeCommand("wqa")

	if !a.quit {
		t.Error(":wqa should quit after saving all buffers")
	}
	if a.buffers[0].buf.Dirty || a.buffers[1].buf.Dirty {
		t.Error(":wqa should save all dirty buffers")
	}

	// Verify files were written.
	data1, err1 := os.ReadFile(path1)
	data2, err2 := os.ReadFile(path2)
	if err1 != nil || err2 != nil {
		t.Fatalf("files should be saved: %v, %v", err1, err2)
	}
	if string(data1) != "content1\n" || string(data2) != "content2\n" {
		t.Errorf("saved content: %q, %q", string(data1), string(data2))
	}
}

func TestCommandWriteQuitAllAlternate(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "file1.txt")

	a := newTestApp(path1)
	a.currentBuf().buf.Lines = []string{"content"}
	a.currentBuf().buf.Dirty = true

	a.executeCommand("qwa")

	if !a.quit {
		t.Error(":qwa should quit after saving")
	}
	if a.currentBuf().buf.Dirty {
		t.Error(":qwa should save dirty buffer")
	}
}

func TestCommandWriteQuitAllWithUnnamed(t *testing.T) {
	a := newTestApp("")
	a.buffers = append(a.buffers, NewEditorBuffer("file2.txt"))

	// Make unnamed buffer dirty.
	a.buffers[0].buf.Lines = []string{"unsaved"}
	a.buffers[0].buf.Dirty = true

	a.executeCommand("wqa")

	if a.quit {
		t.Error(":wqa should not quit with unnamed dirty buffer")
	}
	if a.statusBar.StatusMessage == "" {
		t.Error(":wqa with unnamed dirty buffer should show error message")
	}
	if !strings.Contains(a.statusBar.StatusMessage, "unnamed") {
		t.Errorf("error should mention unnamed buffers, got: %q", a.statusBar.StatusMessage)
	}
}

func TestCommandWriteQuitAllPartialFailure(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.txt")
	invalidPath := "/nonexistent/invalid.txt"

	a := newTestApp(validPath)
	a.buffers = append(a.buffers, NewEditorBuffer(invalidPath))

	a.buffers[0].buf.Lines = []string{"content1"}
	a.buffers[0].buf.Dirty = true
	a.buffers[1].buf.Lines = []string{"content2"}
	a.buffers[1].buf.Dirty = true

	a.executeCommand("wqa")

	if a.quit {
		t.Error(":wqa should not quit if any save fails")
	}
	if a.statusBar.StatusMessage == "" {
		t.Error(":wqa with save failure should show error message")
	}
}
