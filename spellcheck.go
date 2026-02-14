package main

import (
	_ "embed"
	"strings"
	"unicode"

	"github.com/sajari/fuzzy"
)

//go:embed dictionaries/en_GB-large.txt
var dictionaryData string

// SpellError represents a misspelled word location in the buffer
type SpellError struct {
	Line     int    // Buffer line number
	StartCol int    // Starting column (rune index)
	EndCol   int    // Ending column (rune index)
	Word     string // The misspelled word
}

// SpellChecker provides spell checking functionality using a fuzzy model
type SpellChecker struct {
	model *fuzzy.Model
}

// NewSpellChecker creates a new spell checker with the embedded British English dictionary
func NewSpellChecker() (*SpellChecker, error) {
	model := fuzzy.NewModel()

	// Set depth to 2 for better performance vs accuracy trade-off
	model.SetDepth(2)

	// Load words from embedded dictionary
	lines := strings.Split(dictionaryData, "\n")
	for _, word := range lines {
		word = strings.TrimSpace(word)
		if word != "" {
			model.TrainWord(word)
		}
	}

	return &SpellChecker{model: model}, nil
}

// CheckWord returns true if the word is spelled correctly
func (sc *SpellChecker) CheckWord(word string) bool {
	if word == "" {
		return true
	}

	// Convert to lowercase for checking
	lowerWord := strings.ToLower(word)

	// SpellCheck returns the word if it's in the dictionary, or empty string if not found
	correction := sc.model.SpellCheck(lowerWord)

	// If the correction is non-empty and matches the original word, it's spelled correctly
	// Empty string means the word is not in the dictionary
	return correction != "" && correction == lowerWord
}

// wordPosition represents a word and its position in a line
type wordPosition struct {
	word     string
	startCol int
	endCol   int
}

// ExtractWords tokenizes a line into words with their positions (rune indices)
// Words are defined as sequences of letters and apostrophes
func ExtractWords(line string) []wordPosition {
	var words []wordPosition
	runes := []rune(line)

	inWord := false
	var startCol int
	var currentWord strings.Builder

	for i, r := range runes {
		isLetter := unicode.IsLetter(r)
		isApostrophe := r == '\''

		if isLetter || (isApostrophe && inWord) {
			if !inWord {
				// Start of a new word
				startCol = i
				inWord = true
				currentWord.Reset()
			}
			currentWord.WriteRune(r)
		} else {
			if inWord {
				// End of word
				words = append(words, wordPosition{
					word:     currentWord.String(),
					startCol: startCol,
					endCol:   i,
				})
				inWord = false
			}
		}
	}

	// Handle word at end of line
	if inWord {
		words = append(words, wordPosition{
			word:     currentWord.String(),
			startCol: startCol,
			endCol:   len(runes),
		})
	}

	return words
}

// CheckLine checks a line for spelling errors and returns a slice of SpellError
func (sc *SpellChecker) CheckLine(lineNum int, line string) []SpellError {
	var errors []SpellError

	words := ExtractWords(line)
	for _, wp := range words {
		// Skip very short words (1-2 letters) as fuzzy matching doesn't work well for them
		// and they're rarely misspelled anyway
		wordRunes := []rune(wp.word)
		if len(wordRunes) <= 2 {
			continue
		}

		// Skip words that are all uppercase (likely acronyms like API, HTTP)
		allUpper := true
		for _, r := range wordRunes {
			if unicode.IsLetter(r) && !unicode.IsUpper(r) {
				allUpper = false
				break
			}
		}
		if allUpper {
			continue
		}

		// Check spelling
		if !sc.CheckWord(wp.word) {
			errors = append(errors, SpellError{
				Line:     lineNum,
				StartCol: wp.startCol,
				EndCol:   wp.endCol,
				Word:     wp.word,
			})
		}
	}

	return errors
}
