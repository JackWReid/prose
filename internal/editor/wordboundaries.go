package editor

import "unicode"

// WordBoundary represents a word location in the buffer for navigation.
type WordBoundary struct {
	Line     int
	StartCol int
	EndCol   int
}

// FindWordBoundaries scans the entire buffer and returns all word boundaries.
func FindWordBoundaries(buf *Buffer) []WordBoundary {
	var boundaries []WordBoundary
	for lineNum := 0; lineNum < len(buf.Lines); lineNum++ {
		lineBoundaries := extractWordBoundariesFromLine(lineNum, buf.Lines[lineNum])
		boundaries = append(boundaries, lineBoundaries...)
	}
	return boundaries
}

func extractWordBoundariesFromLine(lineNum int, line string) []WordBoundary {
	var boundaries []WordBoundary
	runes := []rune(line)
	inWord := false
	var startCol int

	for i, r := range runes {
		isWordChar := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		if isWordChar {
			if !inWord {
				startCol = i
				inWord = true
			}
		} else {
			if inWord {
				boundaries = append(boundaries, WordBoundary{
					Line: lineNum, StartCol: startCol, EndCol: i,
				})
				inWord = false
			}
		}
	}
	if inWord {
		boundaries = append(boundaries, WordBoundary{
			Line: lineNum, StartCol: startCol, EndCol: len(runes),
		})
	}
	return boundaries
}
