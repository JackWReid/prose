package main

import (
	"fmt"
	"path/filepath"
)

// PromptType indicates what kind of prompt is active.
type PromptType int

const (
	PromptNone    PromptType = iota
	PromptSaveNew                    // "Save as: " for unnamed buffer on first save
	PromptCommand                    // ":" command input
)

// StatusBar generates status bar text and handles prompt state.
type StatusBar struct {
	Prompt        PromptType
	PromptText    string // User input during rename/save-as prompts.
	StatusMessage string // Temporary message (e.g. error from command mode).
}

func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

// FormatLeft returns the left-aligned portion of the status bar.
// bufferInfo is an optional "[2/3]" indicator when multiple buffers are open.
func (s *StatusBar) FormatLeft(filename string, dirty bool, bufferInfo string) string {
	if s.Prompt == PromptSaveNew {
		return fmt.Sprintf(" Save as: %s", s.PromptText)
	}
	if s.Prompt == PromptCommand {
		return fmt.Sprintf(" :%s", s.PromptText)
	}

	if s.StatusMessage != "" {
		return " " + s.StatusMessage
	}

	name := truncatePath(filename)

	// Colour dirty filenames yellow/bold via ANSI codes.
	if dirty {
		name = "\x1b[1;33m" + name + "\x1b[0m\x1b[7m"
	}

	if bufferInfo != "" {
		return fmt.Sprintf(" %s %s", name, bufferInfo)
	}
	return fmt.Sprintf(" %s", name)
}

// FormatRight returns the right-aligned portion of the status bar.
func (s *StatusBar) FormatRight(mode Mode, wordCount int) string {
	if s.Prompt != PromptNone {
		return ""
	}
	modeStr := ""
	switch mode {
	case ModeDefault:
		modeStr = "DEFAULT"
	case ModeEdit:
		modeStr = "EDIT"
	}
	return fmt.Sprintf("%d words  %s ", wordCount, modeStr)
}

// StartPrompt begins a prompt of the given type.
func (s *StatusBar) StartPrompt(pt PromptType) {
	s.Prompt = pt
	s.PromptText = ""
}

// ClearPrompt resets the prompt state.
func (s *StatusBar) ClearPrompt() {
	s.Prompt = PromptNone
	s.PromptText = ""
}

// SetMessage sets a temporary status message.
func (s *StatusBar) SetMessage(msg string) {
	s.StatusMessage = msg
}

// ClearMessage clears the temporary status message.
func (s *StatusBar) ClearMessage() {
	s.StatusMessage = ""
}

// truncatePath shortens a file path to parent/basename.
func truncatePath(filename string) string {
	if filename == "" {
		return "[unnamed]"
	}
	dir := filepath.Base(filepath.Dir(filename))
	base := filepath.Base(filename)
	if dir == "." || dir == "/" {
		return base
	}
	return dir + "/" + base
}

// HandlePromptKey processes a keypress during an active prompt.
// Returns (input string, done bool, cancelled bool).
func (s *StatusBar) HandlePromptKey(key Key) (string, bool, bool) {
	switch key.Type {
	case KeyEscape:
		s.ClearPrompt()
		return "", false, true
	case KeyEnter:
		text := s.PromptText
		s.ClearPrompt()
		return text, true, false
	case KeyBackspace:
		if len(s.PromptText) > 0 {
			runes := []rune(s.PromptText)
			s.PromptText = string(runes[:len(runes)-1])
		}
		return "", false, false
	case KeyRune:
		s.PromptText += string(key.Rune)
		return "", false, false
	}
	return "", false, false
}
