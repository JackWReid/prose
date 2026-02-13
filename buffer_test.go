package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBuffer(t *testing.T) {
	buf := NewBuffer("")
	if len(buf.Lines) != 1 || buf.Lines[0] != "" {
		t.Errorf("new buffer should have one empty line, got %v", buf.Lines)
	}
	if buf.Dirty {
		t.Error("new buffer should not be dirty")
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello\nworld\n"), 0644)

	buf := NewBuffer(path)
	if err := buf.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(buf.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(buf.Lines), buf.Lines)
	}
	if buf.Lines[0] != "hello" || buf.Lines[1] != "world" {
		t.Errorf("unexpected content: %v", buf.Lines)
	}

	buf.InsertChar(0, 5, '!')
	if !buf.Dirty {
		t.Error("buffer should be dirty after edit")
	}

	savePath := filepath.Join(dir, "out.txt")
	if err := buf.Save(savePath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, _ := os.ReadFile(savePath)
	if string(data) != "hello!\nworld\n" {
		t.Errorf("saved content: %q", string(data))
	}
	if buf.Dirty {
		t.Error("buffer should not be dirty after save")
	}
}

func TestLoadNonexistent(t *testing.T) {
	buf := NewBuffer("/tmp/nonexistent_col_test_file.txt")
	if err := buf.Load(); err != nil {
		t.Fatalf("Load nonexistent should not error, got: %v", err)
	}
	if len(buf.Lines) != 1 || buf.Lines[0] != "" {
		t.Errorf("expected single empty line for new file, got %v", buf.Lines)
	}
}

func TestInsertChar(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}

	buf.InsertChar(0, 0, 'H')
	if buf.Lines[0] != "Hhello" {
		t.Errorf("insert at 0: %q", buf.Lines[0])
	}

	buf.InsertChar(0, 6, '!')
	if buf.Lines[0] != "Hhello!" {
		t.Errorf("insert at end: %q", buf.Lines[0])
	}

	buf.InsertChar(0, 3, '-')
	if buf.Lines[0] != "Hhe-llo!" {
		t.Errorf("insert in middle: %q", buf.Lines[0])
	}
}

func TestDeleteChar(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}

	ch, joined := buf.DeleteChar(0, 5)
	if ch != 'o' || joined {
		t.Errorf("delete last char: ch=%c, joined=%v", ch, joined)
	}
	if buf.Lines[0] != "hell" {
		t.Errorf("after delete: %q", buf.Lines[0])
	}

	ch, joined = buf.DeleteChar(0, 1)
	if ch != 'h' || joined {
		t.Errorf("delete first char: ch=%c, joined=%v", ch, joined)
	}
	if buf.Lines[0] != "ell" {
		t.Errorf("after delete: %q", buf.Lines[0])
	}
}

func TestDeleteCharJoinsLines(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello", "world"}

	ch, joined := buf.DeleteChar(1, 0)
	if ch != '\n' || !joined {
		t.Errorf("expected join: ch=%c, joined=%v", ch, joined)
	}
	if len(buf.Lines) != 1 || buf.Lines[0] != "helloworld" {
		t.Errorf("after join: %v", buf.Lines)
	}
}

func TestInsertNewline(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"helloworld"}

	buf.InsertNewline(0, 5)
	if len(buf.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(buf.Lines))
	}
	if buf.Lines[0] != "hello" || buf.Lines[1] != "world" {
		t.Errorf("after split: %v", buf.Lines)
	}
}

func TestInsertNewlineAtStart(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}

	buf.InsertNewline(0, 0)
	if len(buf.Lines) != 2 || buf.Lines[0] != "" || buf.Lines[1] != "hello" {
		t.Errorf("split at start: %v", buf.Lines)
	}
}

func TestInsertNewlineAtEnd(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello"}

	buf.InsertNewline(0, 5)
	if len(buf.Lines) != 2 || buf.Lines[0] != "hello" || buf.Lines[1] != "" {
		t.Errorf("split at end: %v", buf.Lines)
	}
}

func TestLineLen(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello", "日本語"}

	if buf.LineLen(0) != 5 {
		t.Errorf("expected 5, got %d", buf.LineLen(0))
	}
	if buf.LineLen(1) != 3 {
		t.Errorf("expected 3 for Japanese, got %d", buf.LineLen(1))
	}
}

func TestUnicodeInsertDelete(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"café"}

	buf.InsertChar(0, 4, '!')
	if buf.Lines[0] != "café!" {
		t.Errorf("unicode insert: %q", buf.Lines[0])
	}

	ch, _ := buf.DeleteChar(0, 4)
	if ch != 'é' {
		t.Errorf("expected é, got %c", ch)
	}
	if buf.Lines[0] != "caf!" {
		t.Errorf("after unicode delete: %q", buf.Lines[0])
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	os.WriteFile(path, []byte(""), 0644)

	buf := NewBuffer(path)
	buf.Load()
	if len(buf.Lines) != 1 || buf.Lines[0] != "" {
		t.Errorf("empty file should give one empty line, got %v", buf.Lines)
	}
}

func TestLoadNoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notrl.txt")
	os.WriteFile(path, []byte("line1\nline2"), 0644)

	buf := NewBuffer(path)
	buf.Load()
	if len(buf.Lines) != 2 || buf.Lines[0] != "line1" || buf.Lines[1] != "line2" {
		t.Errorf("unexpected: %v", buf.Lines)
	}
}

func TestWordCount(t *testing.T) {
	buf := NewBuffer("")
	buf.Lines = []string{"hello world", "foo bar baz", ""}
	if got := buf.WordCount(); got != 5 {
		t.Errorf("expected 5, got %d", got)
	}

	buf.Lines = []string{""}
	if got := buf.WordCount(); got != 0 {
		t.Errorf("empty buffer: expected 0, got %d", got)
	}

	buf.Lines = []string{"one"}
	if got := buf.WordCount(); got != 1 {
		t.Errorf("single word: expected 1, got %d", got)
	}
}

func TestSaveAddsTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	buf := NewBuffer(path)
	buf.Lines = []string{"hello"}
	buf.Save("")

	data, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("saved file should end with newline")
	}
}
