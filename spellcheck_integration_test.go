package main

import (
	"testing"
)

// TestSpellCheckIntegration verifies the end-to-end spell checking flow
func TestSpellCheckIntegration(t *testing.T) {
	// Initialize spell checker
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("Failed to initialize spell checker: %v", err)
	}

	// Create an editor buffer for a markdown file
	eb := NewEditorBuffer("test.md")
	eb.buf.Lines = []string{
		"This is a test with mispelled words.",
		"The word recieve is wrong.",
		"British spellings like colour are correct.",
		"API and HTTP should be ignored.",
	}

	// Verify the buffer should be spell checked
	if !eb.ShouldSpellCheck() {
		t.Error("Markdown files should be spell checked")
	}

	// Manually trigger spell checking (simulating what PerformSpellCheck does)
	eb.spellErrors = nil
	for i := 0; i < len(eb.buf.Lines); i++ {
		lineErrors := sc.CheckLine(i, eb.buf.Lines[i])
		eb.spellErrors = append(eb.spellErrors, lineErrors...)
	}

	// Verify errors were found
	if len(eb.spellErrors) == 0 {
		t.Error("Expected spell errors to be found")
	}

	// Verify specific errors
	foundMispelled := false
	foundRecieve := false
	for _, err := range eb.spellErrors {
		if err.Word == "mispelled" {
			foundMispelled = true
		}
		if err.Word == "recieve" {
			foundRecieve = true
		}
	}

	if !foundMispelled {
		t.Error("Expected to find 'mispelled' as an error")
	}
	if !foundRecieve {
		t.Error("Expected to find 'recieve' as an error")
	}

	// Verify correct words are not flagged
	for _, err := range eb.spellErrors {
		if err.Word == "colour" {
			t.Errorf("British spelling 'colour' should not be flagged as error")
		}
		if err.Word == "API" || err.Word == "HTTP" {
			t.Errorf("Acronym %q should not be flagged as error", err.Word)
		}
	}

	// Verify spell error count
	if eb.SpellErrorCount() != len(eb.spellErrors) {
		t.Errorf("SpellErrorCount() = %d, want %d", eb.SpellErrorCount(), len(eb.spellErrors))
	}
}

// TestSpellCheckFileTypes verifies that only .md and .txt files are spell checked
func TestSpellCheckFileTypes(t *testing.T) {
	tests := []struct {
		filename     string
		shouldCheck bool
	}{
		{"test.md", true},
		{"README.md", true},
		{"test.markdown", true},
		{"notes.txt", true},
		{"main.go", false},
		{"style.css", false},
		{"script.js", false},
		{"", false}, // Unnamed buffer
	}

	for _, tt := range tests {
		eb := NewEditorBuffer(tt.filename)
		got := eb.ShouldSpellCheck()
		if got != tt.shouldCheck {
			t.Errorf("ShouldSpellCheck(%q) = %v, want %v", tt.filename, got, tt.shouldCheck)
		}
	}
}

// TestSpellCheckDebounce verifies that spell checking is debounced
func TestSpellCheckDebounce(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("Failed to initialize spell checker: %v", err)
	}

	eb := NewEditorBuffer("test.md")
	eb.buf.Lines = []string{"test wrold"}

	// Schedule a spell check
	eb.ScheduleSpellCheck()
	if !eb.spellCheckPending {
		t.Error("Spell check should be pending after scheduling")
	}

	// Immediately try to perform spell check (should not run due to debounce)
	eb.PerformSpellCheck(sc)
	if !eb.spellCheckPending {
		t.Error("Spell check should still be pending (debounced)")
	}
	if len(eb.spellErrors) > 0 {
		t.Error("Spell check should not have run yet due to debounce")
	}
}

// TestSpellCheckBritishSpellings verifies British English spellings are accepted
func TestSpellCheckBritishSpellings(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("Failed to initialize spell checker: %v", err)
	}

	britishWords := []string{
		"colour", "honour", "flavour", "favour",
		"centre", "theatre", "metre", "litre",
		"organise", "realise", "analyse", "recognise",
	}

	for _, word := range britishWords {
		if !sc.CheckWord(word) {
			t.Errorf("British spelling %q should be correct", word)
		}
	}
}

// TestSpellCheckShortWords verifies that short words (1-2 letters) are skipped
func TestSpellCheckShortWords(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("Failed to initialize spell checker: %v", err)
	}

	// Check a line with short words
	errors := sc.CheckLine(0, "I am at is on go by to")
	if len(errors) > 0 {
		t.Errorf("Short words should be skipped, but found errors: %v", errors)
	}
}

// TestSpellCheckContractions verifies contractions work correctly
func TestSpellCheckContractions(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("Failed to initialize spell checker: %v", err)
	}

	contractions := []string{"don't", "can't", "won't", "shouldn't", "wouldn't", "couldn't"}

	for _, word := range contractions {
		if !sc.CheckWord(word) {
			t.Errorf("Contraction %q should be correct", word)
		}
	}
}
