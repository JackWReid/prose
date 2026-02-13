package main

import "testing"

func TestNewEditorBuffer(t *testing.T) {
	eb := NewEditorBuffer("test.md")
	if eb.Filename() != "test.md" {
		t.Errorf("Filename() = %q, want %q", eb.Filename(), "test.md")
	}
	if eb.IsDirty() {
		t.Error("new buffer should not be dirty")
	}
	if eb.WordCount() != 0 {
		t.Errorf("WordCount() = %d, want 0", eb.WordCount())
	}
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("cursor should start at (0,0), got (%d,%d)", eb.cursorLine, eb.cursorCol)
	}
	if eb.scrollOffset != 0 {
		t.Errorf("scrollOffset should start at 0, got %d", eb.scrollOffset)
	}
}

func TestEditorBufferUnnamed(t *testing.T) {
	eb := NewEditorBuffer("")
	if eb.Filename() != "" {
		t.Errorf("Filename() = %q, want empty", eb.Filename())
	}
}

func TestEditorBufferDirty(t *testing.T) {
	eb := NewEditorBuffer("test.txt")
	eb.buf.InsertChar(0, 0, 'a')
	if !eb.IsDirty() {
		t.Error("buffer should be dirty after insert")
	}
}

func TestEditorBufferWordCount(t *testing.T) {
	eb := NewEditorBuffer("test.txt")
	eb.buf.Lines = []string{"hello world", "foo bar baz"}
	if got := eb.WordCount(); got != 5 {
		t.Errorf("WordCount() = %d, want 5", got)
	}
}

func TestEditorBufferHighlighter(t *testing.T) {
	md := NewEditorBuffer("notes.md")
	if _, ok := md.highlighter.(MarkdownHighlighter); !ok {
		t.Errorf("expected MarkdownHighlighter for .md, got %T", md.highlighter)
	}

	plain := NewEditorBuffer("code.go")
	if _, ok := plain.highlighter.(PlainHighlighter); !ok {
		t.Errorf("expected PlainHighlighter for .go, got %T", plain.highlighter)
	}
}
