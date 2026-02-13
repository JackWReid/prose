package main

import (
	"strings"
	"testing"
)

func TestFormatLeftFilename(t *testing.T) {
	sb := NewStatusBar()

	got := sb.FormatLeft("test.txt", false, "")
	if got != " test.txt" {
		t.Errorf("got %q", got)
	}

	got = sb.FormatLeft("test.txt", true, "")
	// Dirty filename should contain yellow/bold ANSI code.
	if !strings.Contains(got, "\x1b[1;33m") {
		t.Errorf("dirty: expected yellow/bold ANSI, got %q", got)
	}
	if !strings.Contains(got, "test.txt") {
		t.Errorf("dirty: should contain filename, got %q", got)
	}

	got = sb.FormatLeft("", false, "")
	if got != " [unnamed]" {
		t.Errorf("unnamed: %q", got)
	}

	// Full path should be truncated to parent/base.
	got = sb.FormatLeft("/Users/jack/Developer/prose/main.go", false, "")
	if got != " prose/main.go" {
		t.Errorf("truncated path: %q", got)
	}
}

func TestFormatLeftBufferInfo(t *testing.T) {
	sb := NewStatusBar()

	got := sb.FormatLeft("test.txt", false, "[2/3]")
	if !strings.Contains(got, "test.txt") || !strings.Contains(got, "[2/3]") {
		t.Errorf("buffer info: %q", got)
	}

	// No buffer info for single buffer.
	got = sb.FormatLeft("test.txt", false, "")
	if strings.Contains(got, "[") {
		t.Errorf("single buffer should have no indicator: %q", got)
	}
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "[unnamed]"},
		{"main.go", "main.go"},
		{"/Users/jack/Developer/prose/main.go", "prose/main.go"},
		{"prose/main.go", "prose/main.go"},
		{"/main.go", "main.go"},
	}
	for _, tt := range tests {
		got := truncatePath(tt.input)
		if got != tt.want {
			t.Errorf("truncatePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStatusMessage(t *testing.T) {
	sb := NewStatusBar()
	sb.SetMessage("Error: unsaved changes")
	got := sb.FormatLeft("test.txt", false, "")
	if got != " Error: unsaved changes" {
		t.Errorf("status message: %q", got)
	}
	sb.ClearMessage()
	got = sb.FormatLeft("test.txt", false, "")
	if got != " test.txt" {
		t.Errorf("after clear: %q", got)
	}
}

func TestFormatLeftPrompt(t *testing.T) {
	sb := NewStatusBar()
	sb.StartPrompt(PromptSaveNew)
	sb.PromptText = "foo.txt"

	got := sb.FormatLeft("test.txt", false, "")
	if got != " Save as: foo.txt" {
		t.Errorf("save-new prompt: %q", got)
	}

	sb.StartPrompt(PromptCommand)
	sb.PromptText = "wq"
	got = sb.FormatLeft("test.txt", true, "")
	if got != " :wq" {
		t.Errorf("command prompt: %q", got)
	}
}

func TestFormatRight(t *testing.T) {
	sb := NewStatusBar()
	if got := sb.FormatRight(ModeDefault, 42); got != "42 words  DEFAULT " {
		t.Errorf("default mode: %q", got)
	}
	if got := sb.FormatRight(ModeEdit, 0); got != "0 words  EDIT " {
		t.Errorf("edit mode: %q", got)
	}
	sb.StartPrompt(PromptSaveNew)
	if got := sb.FormatRight(ModeDefault, 10); got != "" {
		t.Errorf("during prompt: %q", got)
	}
}

func TestHandlePromptKeyInput(t *testing.T) {
	sb := NewStatusBar()
	sb.StartPrompt(PromptCommand)

	sb.HandlePromptKey(Key{Type: KeyRune, Rune: 'a'})
	sb.HandlePromptKey(Key{Type: KeyRune, Rune: 'b'})
	if sb.PromptText != "ab" {
		t.Errorf("prompt text: %q", sb.PromptText)
	}

	sb.HandlePromptKey(Key{Type: KeyBackspace})
	if sb.PromptText != "a" {
		t.Errorf("after backspace: %q", sb.PromptText)
	}

	text, done, cancelled := sb.HandlePromptKey(Key{Type: KeyEnter})
	if !done || cancelled || text != "a" {
		t.Errorf("enter: text=%q, done=%v, cancelled=%v", text, done, cancelled)
	}
	if sb.Prompt != PromptNone {
		t.Error("prompt should be cleared after enter")
	}
}

func TestHandlePromptKeyCancel(t *testing.T) {
	sb := NewStatusBar()
	sb.StartPrompt(PromptCommand)
	sb.HandlePromptKey(Key{Type: KeyRune, Rune: 'x'})

	_, _, cancelled := sb.HandlePromptKey(Key{Type: KeyEscape})
	if !cancelled {
		t.Error("escape should cancel")
	}
	if sb.Prompt != PromptNone {
		t.Error("prompt should be cleared after escape")
	}
}
