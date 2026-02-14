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
	PromptSearch                     // "/" search input
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
// spellErrorCount is the number of spelling errors in the buffer.
func (s *StatusBar) FormatLeft(filename string, dirty bool, bufferInfo string, spellErrorCount int, isScratch bool) string {
	if s.Prompt == PromptSaveNew {
		return fmt.Sprintf(" Save as: %s", s.PromptText)
	}
	if s.Prompt == PromptCommand {
		return fmt.Sprintf(" :%s", s.PromptText)
	}
	if s.Prompt == PromptSearch {
		return fmt.Sprintf(" /%s", s.PromptText)
	}

	if s.StatusMessage != "" {
		return " " + s.StatusMessage
	}

	name := truncatePathScratch(filename, isScratch)

	// Colour dirty filenames bold + darker orange via ANSI codes.
	// In reverse video mode, use background code to set text color.
	if dirty {
		name = "\x1b[1;48;5;208m" + name + "\x1b[22;49m"
	}

	// Add spell error indicator (red dot) if there are errors
	// In reverse video mode, background codes affect foreground and vice versa
	// So we use background code (48) to get red text in the inverted status bar
	spellIndicator := ""
	if spellErrorCount > 0 {
		spellIndicator = " \x1b[48;5;9m●\x1b[49m"
	}

	if bufferInfo != "" {
		return fmt.Sprintf(" %s%s %s", name, spellIndicator, bufferInfo)
	}
	return fmt.Sprintf(" %s%s", name, spellIndicator)
}

// FormatRight returns the right-aligned portion of the status bar.
func (s *StatusBar) FormatRight(mode Mode, wordCount int, spellErrorCount int, searchActive bool, searchCurrentIdx int, searchMatchCount int) string {
	if s.Prompt != PromptNone {
		return ""
	}
	modeStr := ""
	switch mode {
	case ModeDefault:
		modeStr = "DEFAULT"
	case ModeEdit:
		modeStr = "EDIT"
	case ModeLineSelect:
		modeStr = "LINE-SELECT"
	}

	// Show search match counter if search is active
	searchStr := ""
	if searchActive && searchMatchCount > 0 {
		searchStr = fmt.Sprintf("%d/%d matches  ", searchCurrentIdx+1, searchMatchCount)
	}

	// Show error count if there are spelling errors
	errorStr := ""
	if spellErrorCount > 0 {
		errorStr = fmt.Sprintf("%d errors  ", spellErrorCount)
	}

	return fmt.Sprintf("%s%s%d words  %s ", searchStr, errorStr, wordCount, modeStr)
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

// truncatePathScratch is like truncatePath but handles scratch buffers.
func truncatePathScratch(filename string, isScratch bool) string {
	if isScratch {
		return "[scratch]"
	}
	return truncatePath(filename)
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
