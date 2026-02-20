package spell

import (
	"testing"
)

func TestNewSpellChecker(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("NewSpellChecker() failed: %v", err)
	}
	if sc == nil || sc.model == nil {
		t.Fatal("NewSpellChecker() returned nil model")
	}
}

func TestCheckWord(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("NewSpellChecker() failed: %v", err)
	}

	tests := []struct {
		word     string
		expected bool
		desc     string
	}{
		// Correct British spellings
		{"colour", true, "British spelling of color"},
		{"honour", true, "British spelling of honor"},
		{"organise", true, "British spelling with -ise"},
		{"theatre", true, "British spelling of theater"},
		{"centre", true, "British spelling of center"},

		// Common correct words
		{"hello", true, "common word"},
		{"world", true, "common word"},
		{"the", true, "common article"},
		{"test", true, "common word"},

		// Correct words with capitals (should normalize)
		{"Hello", true, "capitalized word"},
		{"WORLD", true, "all caps word"},

		// Contractions
		{"don't", true, "contraction with apostrophe"},
		{"can't", true, "contraction with apostrophe"},
		{"won't", true, "contraction with apostrophe"},

		// Misspelled words
		{"helllo", false, "misspelled hello"},
		{"wrold", false, "misspelled world"},
		{"teh", false, "misspelled the"},
		{"speling", false, "misspelled spelling"},
		{"recieve", false, "common misspelling of receive"},
		{"definately", false, "common misspelling of definitely"},

		// Edge cases
		{"", true, "empty string should return true"},
	}

	for _, tt := range tests {
		result := sc.CheckWord(tt.word)
		if result != tt.expected {
			t.Errorf("CheckWord(%q) = %v, expected %v (%s)", tt.word, result, tt.expected, tt.desc)
		}
	}
}

func TestExtractWords(t *testing.T) {
	tests := []struct {
		line     string
		expected []wordPosition
		desc     string
	}{
		{
			line: "hello world",
			expected: []wordPosition{
				{word: "hello", startCol: 0, endCol: 5},
				{word: "world", startCol: 6, endCol: 11},
			},
			desc: "simple two words",
		},
		{
			line: "don't can't won't",
			expected: []wordPosition{
				{word: "don't", startCol: 0, endCol: 5},
				{word: "can't", startCol: 6, endCol: 11},
				{word: "won't", startCol: 12, endCol: 17},
			},
			desc: "contractions with apostrophes",
		},
		{
			line: "word123 test-case under_score",
			expected: []wordPosition{
				{word: "word", startCol: 0, endCol: 4},
				{word: "test", startCol: 8, endCol: 12},
				{word: "case", startCol: 13, endCol: 17},
				{word: "under", startCol: 18, endCol: 23},
				{word: "score", startCol: 24, endCol: 29},
			},
			desc: "words with numbers and punctuation",
		},
		{
			line: "  leading  trailing  ",
			expected: []wordPosition{
				{word: "leading", startCol: 2, endCol: 9},
				{word: "trailing", startCol: 11, endCol: 19},
			},
			desc: "extra whitespace",
		},
		{
			line:     "",
			expected: []wordPosition{},
			desc:     "empty line",
		},
		{
			line:     "123 456 789",
			expected: []wordPosition{},
			desc:     "only numbers",
		},
		{
			line: "a I",
			expected: []wordPosition{
				{word: "a", startCol: 0, endCol: 1},
				{word: "I", startCol: 2, endCol: 3},
			},
			desc: "single letter words",
		},
	}

	for _, tt := range tests {
		result := ExtractWords(tt.line)
		if len(result) != len(tt.expected) {
			t.Errorf("ExtractWords(%q) returned %d words, expected %d (%s)",
				tt.line, len(result), len(tt.expected), tt.desc)
			continue
		}

		for i, wp := range result {
			exp := tt.expected[i]
			if wp.word != exp.word || wp.startCol != exp.startCol || wp.endCol != exp.endCol {
				t.Errorf("ExtractWords(%q)[%d] = {%q, %d, %d}, expected {%q, %d, %d} (%s)",
					tt.line, i, wp.word, wp.startCol, wp.endCol,
					exp.word, exp.startCol, exp.endCol, tt.desc)
			}
		}
	}
}

func TestCheckLine(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("NewSpellChecker() failed: %v", err)
	}

	tests := []struct {
		line        string
		expectCount int
		desc        string
	}{
		{
			line:        "hello world",
			expectCount: 0,
			desc:        "all correct words",
		},
		{
			line:        "helllo wrold",
			expectCount: 2,
			desc:        "two misspelled words",
		},
		{
			line:        "The colour is beautiful",
			expectCount: 0,
			desc:        "British spelling should be correct",
		},
		{
			line:        "This is speling mistake",
			expectCount: 1,
			desc:        "one misspelled word",
		},
		{
			line:        "don't won't can't",
			expectCount: 0,
			desc:        "contractions should be correct",
		},
		{
			line:        "I like API and HTTP",
			expectCount: 0,
			desc:        "acronyms (all caps) should be skipped",
		},
		{
			line:        "",
			expectCount: 0,
			desc:        "empty line",
		},
		{
			line:        "123 456",
			expectCount: 0,
			desc:        "only numbers",
		},
	}

	for _, tt := range tests {
		errors := sc.CheckLine(0, tt.line)
		if len(errors) != tt.expectCount {
			t.Errorf("CheckLine(%q) found %d errors, expected %d (%s)",
				tt.line, len(errors), tt.expectCount, tt.desc)
		}
	}
}

func TestCheckLinePositions(t *testing.T) {
	sc, err := NewSpellChecker()
	if err != nil {
		t.Fatalf("NewSpellChecker() failed: %v", err)
	}

	line := "helllo world wrold test"
	errors := sc.CheckLine(5, line)

	// Should find "helllo" and "wrold" as errors
	if len(errors) != 2 {
		t.Fatalf("CheckLine(%q) found %d errors, expected 2", line, len(errors))
	}

	// Check first error (helllo)
	if errors[0].Word != "helllo" || errors[0].StartCol != 0 || errors[0].EndCol != 6 || errors[0].Line != 5 {
		t.Errorf("First error: got {%q, line=%d, start=%d, end=%d}, expected {%q, line=5, start=0, end=6}",
			errors[0].Word, errors[0].Line, errors[0].StartCol, errors[0].EndCol, "helllo")
	}

	// Check second error (wrold)
	if errors[1].Word != "wrold" || errors[1].StartCol != 13 || errors[1].EndCol != 18 || errors[1].Line != 5 {
		t.Errorf("Second error: got {%q, line=%d, start=%d, end=%d}, expected {%q, line=5, start=13, end=18}",
			errors[1].Word, errors[1].Line, errors[1].StartCol, errors[1].EndCol, "wrold")
	}
}
