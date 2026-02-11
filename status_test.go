package main

import "testing"

func TestFormatLeftFilename(t *testing.T) {
	sb := NewStatusBar()

	got := sb.FormatLeft("test.txt", false)
	if got != " test.txt" {
		t.Errorf("got %q", got)
	}

	got = sb.FormatLeft("test.txt", true)
	if got != " test.txt [+]" {
		t.Errorf("dirty: %q", got)
	}

	got = sb.FormatLeft("", false)
	if got != " [unnamed]" {
		t.Errorf("unnamed: %q", got)
	}
}

func TestFormatLeftPrompt(t *testing.T) {
	sb := NewStatusBar()
	sb.StartPrompt(PromptRename)
	sb.PromptText = "foo.txt"

	got := sb.FormatLeft("test.txt", false)
	if got != " Save as: foo.txt" {
		t.Errorf("rename prompt: %q", got)
	}

	sb.StartPrompt(PromptQuitDirty)
	got = sb.FormatLeft("test.txt", true)
	if got != " Unsaved changes. Press x to discard, or s to save." {
		t.Errorf("quit prompt: %q", got)
	}
}

func TestFormatRight(t *testing.T) {
	sb := NewStatusBar()
	if got := sb.FormatRight(ModeDefault); got != "DEFAULT " {
		t.Errorf("default mode: %q", got)
	}
	if got := sb.FormatRight(ModeEdit); got != "EDIT " {
		t.Errorf("edit mode: %q", got)
	}
	sb.StartPrompt(PromptRename)
	if got := sb.FormatRight(ModeDefault); got != "" {
		t.Errorf("during prompt: %q", got)
	}
}

func TestHandlePromptKeyInput(t *testing.T) {
	sb := NewStatusBar()
	sb.StartPrompt(PromptRename)

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
	sb.StartPrompt(PromptRename)
	sb.HandlePromptKey(Key{Type: KeyRune, Rune: 'x'})

	_, _, cancelled := sb.HandlePromptKey(Key{Type: KeyEscape})
	if !cancelled {
		t.Error("escape should cancel")
	}
	if sb.Prompt != PromptNone {
		t.Error("prompt should be cleared after escape")
	}
}
