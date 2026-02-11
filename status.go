package main

import "fmt"

// PromptType indicates what kind of prompt is active.
type PromptType int

const (
	PromptNone     PromptType = iota
	PromptRename              // "Save as: " filename input
	PromptQuitDirty           // "Unsaved changes. x to discard, s to save."
	PromptSaveNew             // "Save as: " for unnamed buffer on first save
)

// StatusBar generates status bar text and handles prompt state.
type StatusBar struct {
	Prompt     PromptType
	PromptText string // User input during rename/save-as prompts.
}

func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

// FormatLeft returns the left-aligned portion of the status bar.
func (s *StatusBar) FormatLeft(filename string, dirty bool) string {
	if s.Prompt == PromptRename || s.Prompt == PromptSaveNew {
		return fmt.Sprintf(" Save as: %s", s.PromptText)
	}
	if s.Prompt == PromptQuitDirty {
		return " Unsaved changes. Press x to discard, or s to save."
	}

	name := filename
	if name == "" {
		name = "[unnamed]"
	}
	mod := ""
	if dirty {
		mod = " [+]"
	}
	return fmt.Sprintf(" %s%s", name, mod)
}

// FormatRight returns the right-aligned portion of the status bar.
func (s *StatusBar) FormatRight(mode Mode) string {
	if s.Prompt != PromptNone {
		return ""
	}
	switch mode {
	case ModeDefault:
		return "DEFAULT "
	case ModeEdit:
		return "EDIT "
	}
	return ""
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
