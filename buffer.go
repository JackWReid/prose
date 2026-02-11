package main

import (
	"os"
	"strings"
)

// Buffer holds the text content as a slice of lines (hard lines, split on \n).
type Buffer struct {
	Lines    []string
	Dirty    bool
	Filename string
}

func NewBuffer(filename string) *Buffer {
	return &Buffer{
		Lines:    []string{""},
		Filename: filename,
	}
}

// Load reads a file into the buffer.
func (b *Buffer) Load() error {
	if b.Filename == "" {
		return nil
	}
	data, err := os.ReadFile(b.Filename)
	if err != nil {
		if os.IsNotExist(err) {
			// New file — start with empty buffer.
			b.Lines = []string{""}
			return nil
		}
		return err
	}
	text := string(data)
	// Strip trailing newline to avoid a phantom empty line.
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		b.Lines = []string{""}
	} else {
		b.Lines = strings.Split(text, "\n")
	}
	b.Dirty = false
	return nil
}

// Save writes the buffer to the given filename (or current filename).
func (b *Buffer) Save(filename string) error {
	if filename != "" {
		b.Filename = filename
	}
	if b.Filename == "" {
		return nil // Caller should prompt for a name.
	}
	content := strings.Join(b.Lines, "\n") + "\n"
	err := os.WriteFile(b.Filename, []byte(content), 0644)
	if err != nil {
		return err
	}
	b.Dirty = false
	return nil
}

// InsertChar inserts a character at the given line and column position.
func (b *Buffer) InsertChar(line, col int, ch rune) {
	if line < 0 || line >= len(b.Lines) {
		return
	}
	runes := []rune(b.Lines[line])
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:col]...)
	newRunes = append(newRunes, ch)
	newRunes = append(newRunes, runes[col:]...)
	b.Lines[line] = string(newRunes)
	b.Dirty = true
}

// DeleteChar deletes the character before the given position.
// Returns the deleted rune and whether a line join occurred.
func (b *Buffer) DeleteChar(line, col int) (rune, bool) {
	if line < 0 || line >= len(b.Lines) {
		return 0, false
	}
	if col > 0 {
		runes := []rune(b.Lines[line])
		if col > len(runes) {
			col = len(runes)
		}
		ch := runes[col-1]
		newRunes := make([]rune, 0, len(runes)-1)
		newRunes = append(newRunes, runes[:col-1]...)
		newRunes = append(newRunes, runes[col:]...)
		b.Lines[line] = string(newRunes)
		b.Dirty = true
		return ch, false
	}
	// col == 0: join with previous line.
	if line == 0 {
		return 0, false
	}
	b.JoinLines(line - 1)
	b.Dirty = true
	return '\n', true
}

// InsertNewline splits the line at the given position.
func (b *Buffer) InsertNewline(line, col int) {
	if line < 0 || line >= len(b.Lines) {
		return
	}
	runes := []rune(b.Lines[line])
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	before := string(runes[:col])
	after := string(runes[col:])
	b.Lines[line] = before
	// Insert new line after.
	newLines := make([]string, 0, len(b.Lines)+1)
	newLines = append(newLines, b.Lines[:line+1]...)
	newLines = append(newLines, after)
	newLines = append(newLines, b.Lines[line+1:]...)
	b.Lines = newLines
	b.Dirty = true
}

// JoinLines joins line[idx] with line[idx+1].
func (b *Buffer) JoinLines(idx int) {
	if idx < 0 || idx+1 >= len(b.Lines) {
		return
	}
	b.Lines[idx] += b.Lines[idx+1]
	b.Lines = append(b.Lines[:idx+1], b.Lines[idx+2:]...)
	b.Dirty = true
}

// LineLen returns the rune-length of a given line.
func (b *Buffer) LineLen(line int) int {
	if line < 0 || line >= len(b.Lines) {
		return 0
	}
	return len([]rune(b.Lines[line]))
}

// LineCount returns the number of lines.
func (b *Buffer) LineCount() int {
	return len(b.Lines)
}
