package main

import (
	"testing"
)

// TestWordJumpingSameLine verifies word jumping on the same line
func TestWordJumpingSameLine(t *testing.T) {
	app := NewApp([]string{})
	eb := app.currentBuf()
	eb.buf.Lines = []string{"hello world test"}

	// Start at beginning
	eb.cursorLine = 0
	eb.cursorCol = 0

	// Jump to next word (world)
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 6 {
		t.Errorf("After first w: cursor at (%d, %d), expected (0, 6)", eb.cursorLine, eb.cursorCol)
	}

	// Jump to next word (test)
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 12 {
		t.Errorf("After second w: cursor at (%d, %d), expected (0, 12)", eb.cursorLine, eb.cursorCol)
	}

	// Jump back to previous word (world)
	app.jumpToPrevWord()
	if eb.cursorLine != 0 || eb.cursorCol != 6 {
		t.Errorf("After b: cursor at (%d, %d), expected (0, 6)", eb.cursorLine, eb.cursorCol)
	}

	// Jump back to previous word (hello)
	app.jumpToPrevWord()
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("After second b: cursor at (%d, %d), expected (0, 0)", eb.cursorLine, eb.cursorCol)
	}
}

// TestWordJumpingAcrossLines verifies word jumping across multiple lines
func TestWordJumpingAcrossLines(t *testing.T) {
	app := NewApp([]string{})
	eb := app.currentBuf()
	eb.buf.Lines = []string{
		"first line",
		"second line",
		"third line",
	}

	// Start at beginning of first line
	eb.cursorLine = 0
	eb.cursorCol = 0

	// Jump through words
	app.jumpToNextWord() // "line" on first line
	if eb.cursorLine != 0 || eb.cursorCol != 6 {
		t.Errorf("Jump 1: cursor at (%d, %d), expected (0, 6)", eb.cursorLine, eb.cursorCol)
	}

	app.jumpToNextWord() // "second" on second line
	if eb.cursorLine != 1 || eb.cursorCol != 0 {
		t.Errorf("Jump 2: cursor at (%d, %d), expected (1, 0)", eb.cursorLine, eb.cursorCol)
	}

	app.jumpToNextWord() // "line" on second line
	if eb.cursorLine != 1 || eb.cursorCol != 7 {
		t.Errorf("Jump 3: cursor at (%d, %d), expected (1, 7)", eb.cursorLine, eb.cursorCol)
	}

	// Jump back
	app.jumpToPrevWord() // "second" on second line
	if eb.cursorLine != 1 || eb.cursorCol != 0 {
		t.Errorf("Jump back 1: cursor at (%d, %d), expected (1, 0)", eb.cursorLine, eb.cursorCol)
	}

	app.jumpToPrevWord() // "line" on first line
	if eb.cursorLine != 0 || eb.cursorCol != 6 {
		t.Errorf("Jump back 2: cursor at (%d, %d), expected (0, 6)", eb.cursorLine, eb.cursorCol)
	}
}

// TestWordJumpingWrapAround verifies wrap-around behavior
func TestWordJumpingWrapAround(t *testing.T) {
	app := NewApp([]string{})
	eb := app.currentBuf()
	eb.buf.Lines = []string{"first second"}

	// Start at end (after "second")
	eb.cursorLine = 0
	eb.cursorCol = 12

	// Jump forward should wrap to first word
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("Forward wrap: cursor at (%d, %d), expected (0, 0)", eb.cursorLine, eb.cursorCol)
	}

	// Jump backward should wrap to last word
	app.jumpToPrevWord()
	if eb.cursorLine != 0 || eb.cursorCol != 6 {
		t.Errorf("Backward wrap: cursor at (%d, %d), expected (0, 6)", eb.cursorLine, eb.cursorCol)
	}
}

// TestWordJumpingEmptyBuffer verifies behavior on empty buffer
func TestWordJumpingEmptyBuffer(t *testing.T) {
	app := NewApp([]string{})
	eb := app.currentBuf()
	eb.buf.Lines = []string{""}

	eb.cursorLine = 0
	eb.cursorCol = 0

	// Should not crash or move
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("Empty buffer next: cursor moved to (%d, %d)", eb.cursorLine, eb.cursorCol)
	}

	app.jumpToPrevWord()
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("Empty buffer prev: cursor moved to (%d, %d)", eb.cursorLine, eb.cursorCol)
	}
}

// TestWordJumpingVimStyleWords verifies Vim-style word definition
func TestWordJumpingVimStyleWords(t *testing.T) {
	app := NewApp([]string{})
	eb := app.currentBuf()
	eb.buf.Lines = []string{"hello_world hello-world test123"}

	eb.cursorLine = 0
	eb.cursorCol = 0

	// "hello_world" should be one word
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 12 {
		t.Errorf("After underscore word: cursor at (%d, %d), expected (0, 12) for 'hello'", eb.cursorLine, eb.cursorCol)
	}

	// "hello" (from hello-world) should be next
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 18 {
		t.Errorf("After hyphen word part 1: cursor at (%d, %d), expected (0, 18) for '-world'", eb.cursorLine, eb.cursorCol)
	}

	// "world" should be separate
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 24 {
		t.Errorf("After hyphen word part 2: cursor at (%d, %d), expected (0, 24) for 'test123'", eb.cursorLine, eb.cursorCol)
	}
}

// TestWordJumpingSingleWord verifies behavior with single word buffer
func TestWordJumpingSingleWord(t *testing.T) {
	app := NewApp([]string{})
	eb := app.currentBuf()
	eb.buf.Lines = []string{"word"}

	eb.cursorLine = 0
	eb.cursorCol = 0

	// Jump forward should wrap to same word
	app.jumpToNextWord()
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("Single word forward: cursor at (%d, %d), expected (0, 0)", eb.cursorLine, eb.cursorCol)
	}

	// Jump backward should stay at same word
	app.jumpToPrevWord()
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("Single word backward: cursor at (%d, %d), expected (0, 0)", eb.cursorLine, eb.cursorCol)
	}
}

// TestWordJumpingWithEmptyLines verifies behavior with empty lines
func TestWordJumpingWithEmptyLines(t *testing.T) {
	app := NewApp([]string{})
	eb := app.currentBuf()
	eb.buf.Lines = []string{
		"first",
		"",
		"second",
	}

	eb.cursorLine = 0
	eb.cursorCol = 0

	// Jump should skip empty line
	app.jumpToNextWord()
	if eb.cursorLine != 2 || eb.cursorCol != 0 {
		t.Errorf("Skip empty line: cursor at (%d, %d), expected (2, 0)", eb.cursorLine, eb.cursorCol)
	}

	// Jump back should skip empty line
	app.jumpToPrevWord()
	if eb.cursorLine != 0 || eb.cursorCol != 0 {
		t.Errorf("Skip empty line backward: cursor at (%d, %d), expected (0, 0)", eb.cursorLine, eb.cursorCol)
	}
}
