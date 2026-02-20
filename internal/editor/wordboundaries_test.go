package editor

import (
	"testing"
)

func TestExtractWordBoundariesFromLine(t *testing.T) {
	tests := []struct {
		line     string
		expected []WordBoundary
		desc     string
	}{
		{
			line: "hello world",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 5},
				{Line: 0, StartCol: 6, EndCol: 11},
			},
			desc: "simple two words",
		},
		{
			line: "word_with_underscores",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 21},
			},
			desc: "underscores: one word",
		},
		{
			line: "hello-world",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 5},
				{Line: 0, StartCol: 6, EndCol: 11},
			},
			desc: "hyphens: two words",
		},
		{
			line: "hello123world",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 13},
			},
			desc: "mixed alphanumeric: one word",
		},
		{
			line: "test_123_hello",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 14},
			},
			desc: "underscores with numbers: one word",
		},
		{
			line:     "",
			expected: []WordBoundary{},
			desc:     "empty line: zero boundaries",
		},
		{
			line:     "   ",
			expected: []WordBoundary{},
			desc:     "whitespace only: zero boundaries",
		},
		{
			line: "a b c",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 1},
				{Line: 0, StartCol: 2, EndCol: 3},
				{Line: 0, StartCol: 4, EndCol: 5},
			},
			desc: "single letter words",
		},
		{
			line: "  leading  trailing  ",
			expected: []WordBoundary{
				{Line: 0, StartCol: 2, EndCol: 9},
				{Line: 0, StartCol: 11, EndCol: 19},
			},
			desc: "extra whitespace",
		},
		{
			line: "word! @hash #tag",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 4},
				{Line: 0, StartCol: 7, EndCol: 11},
				{Line: 0, StartCol: 13, EndCol: 16},
			},
			desc: "punctuation separates words",
		},
		{
			line: "_underscore_start",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 17},
			},
			desc: "underscore at start",
		},
		{
			line: "123",
			expected: []WordBoundary{
				{Line: 0, StartCol: 0, EndCol: 3},
			},
			desc: "numbers only: one word",
		},
	}

	for _, tt := range tests {
		result := extractWordBoundariesFromLine(0, tt.line)
		if len(result) != len(tt.expected) {
			t.Errorf("extractWordBoundariesFromLine(%q) returned %d boundaries, expected %d (%s)",
				tt.line, len(result), len(tt.expected), tt.desc)
			continue
		}

		for i, wb := range result {
			exp := tt.expected[i]
			if wb.Line != exp.Line || wb.StartCol != exp.StartCol || wb.EndCol != exp.EndCol {
				t.Errorf("extractWordBoundariesFromLine(%q)[%d] = {line=%d, start=%d, end=%d}, expected {line=%d, start=%d, end=%d} (%s)",
					tt.line, i, wb.Line, wb.StartCol, wb.EndCol,
					exp.Line, exp.StartCol, exp.EndCol, tt.desc)
			}
		}
	}
}

func TestFindWordBoundaries(t *testing.T) {
	tests := []struct {
		lines    []string
		expected int
		desc     string
	}{
		{
			lines:    []string{"hello world", "test case"},
			expected: 4,
			desc:     "two lines with two words each",
		},
		{
			lines:    []string{"word_with_underscores", "hello-world"},
			expected: 3,
			desc:     "underscores vs hyphens",
		},
		{
			lines:    []string{""},
			expected: 0,
			desc:     "empty buffer",
		},
		{
			lines:    []string{"one", "", "two"},
			expected: 2,
			desc:     "empty line in middle",
		},
	}

	for _, tt := range tests {
		buf := &Buffer{Lines: tt.lines}
		result := FindWordBoundaries(buf)
		if len(result) != tt.expected {
			t.Errorf("FindWordBoundaries() returned %d boundaries, expected %d (%s)",
				len(result), tt.expected, tt.desc)
		}
	}
}
