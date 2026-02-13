package main

import (
	"os"
	"path/filepath"
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
