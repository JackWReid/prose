package main

import (
	"strings"
	"testing"
)

func TestDetectHighlighterMarkdown(t *testing.T) {
	cases := []struct {
		filename string
		wantType string
	}{
		{"notes.md", "Markdown"},
		{"README.markdown", "Markdown"},
		{"doc.mdx", "Markdown"},
		{"NOTES.MD", "Markdown"},
		{"readme.txt", "Plain"},
		{"code.go", "Plain"},
		{"", "Plain"},
		{"noext", "Plain"},
	}
	for _, tc := range cases {
		h := DetectHighlighter(tc.filename)
		switch tc.wantType {
		case "Markdown":
			if _, ok := h.(MarkdownHighlighter); !ok {
				t.Errorf("DetectHighlighter(%q) = %T, want MarkdownHighlighter", tc.filename, h)
			}
		case "Plain":
			if _, ok := h.(PlainHighlighter); !ok {
				t.Errorf("DetectHighlighter(%q) = %T, want PlainHighlighter", tc.filename, h)
			}
		}
	}
}

func TestPlainHighlighterPassthrough(t *testing.T) {
	h := PlainHighlighter{}
	input := "Hello, **world**!"
	if got := h.Highlight(input); got != input {
		t.Errorf("PlainHighlighter.Highlight(%q) = %q, want unchanged", input, got)
	}
}

func TestMarkdownHeadings(t *testing.T) {
	h := MarkdownHighlighter{}
	cases := []string{
		"# Heading 1",
		"## Heading 2",
		"### Heading 3",
		"###### Heading 6",
	}
	for _, line := range cases {
		got := h.Highlight(line)
		if !strings.HasPrefix(got, "\x1b[1;34m") {
			t.Errorf("heading %q should start with bold blue ANSI code, got %q", line, got)
		}
		if !strings.Contains(got, line) {
			t.Errorf("heading output should contain original text %q", line)
		}
	}
}

func TestMarkdownNotHeading(t *testing.T) {
	h := MarkdownHighlighter{}
	// No space after # — not a heading.
	got := h.Highlight("#nospace")
	if strings.HasPrefix(got, "\x1b[1;34m") {
		t.Error("#nospace should not be treated as a heading")
	}
}

func TestMarkdownBlockquote(t *testing.T) {
	h := MarkdownHighlighter{}
	got := h.Highlight("> a quote")
	if !strings.HasPrefix(got, "\x1b[90m") {
		t.Errorf("blockquote should start with dark grey, got %q", got)
	}
}

func TestMarkdownHorizontalRule(t *testing.T) {
	h := MarkdownHighlighter{}
	for _, hr := range []string{"---", "***", "___", "-----"} {
		got := h.Highlight(hr)
		if !strings.HasPrefix(got, "\x1b[90m") {
			t.Errorf("HR %q should start with dark grey, got %q", hr, got)
		}
	}
}

func TestMarkdownBold(t *testing.T) {
	h := MarkdownHighlighter{}
	got := h.Highlight("some **bold** text")
	if !strings.Contains(got, "\x1b[1;33m") {
		t.Errorf("bold text should contain bold yellow ANSI, got %q", got)
	}
	if !strings.Contains(got, "bold") {
		t.Error("bold text content should be preserved")
	}
}

func TestMarkdownItalic(t *testing.T) {
	h := MarkdownHighlighter{}
	got := h.Highlight("some *italic* text")
	if !strings.Contains(got, "\x1b[3;36m") {
		t.Errorf("italic text should contain italic cyan ANSI, got %q", got)
	}
}

func TestMarkdownInlineCode(t *testing.T) {
	h := MarkdownHighlighter{}
	got := h.Highlight("run `go test` now")
	if !strings.Contains(got, "\x1b[35m") {
		t.Errorf("inline code should contain magenta ANSI, got %q", got)
	}
	if !strings.Contains(got, "go test") {
		t.Error("code content should be preserved")
	}
}

func TestMarkdownLink(t *testing.T) {
	h := MarkdownHighlighter{}
	got := h.Highlight("see [my link](https://example.com) here")
	if !strings.Contains(got, "\x1b[4;32m") {
		t.Errorf("link text should contain underlined green ANSI, got %q", got)
	}
	if !strings.Contains(got, "my link") {
		t.Error("link text should be preserved")
	}
}

func TestMarkdownResetAtEnd(t *testing.T) {
	h := MarkdownHighlighter{}
	// Inline-styled lines should end with reset.
	got := h.Highlight("some **bold** text")
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("highlighted line should end with reset, got %q", got)
	}
}

// --- TruncateVisible tests ---

func TestTruncateVisiblePlainText(t *testing.T) {
	got := TruncateVisible("hello world", 5)
	if got != "hello\x1b[0m" {
		t.Errorf("TruncateVisible plain = %q, want %q", got, "hello\x1b[0m")
	}
}

func TestTruncateVisibleShorterThanMax(t *testing.T) {
	got := TruncateVisible("hi", 10)
	if got != "hi" {
		t.Errorf("TruncateVisible short = %q, want %q", got, "hi")
	}
}

func TestTruncateVisibleExactLength(t *testing.T) {
	got := TruncateVisible("hello", 5)
	// Exactly at limit — no truncation needed, but we hit the >= check.
	if !strings.Contains(got, "hello") {
		t.Errorf("TruncateVisible exact = %q, should contain 'hello'", got)
	}
}

func TestTruncateVisibleWithANSI(t *testing.T) {
	// "AB" with bold around A: \x1b[1mA\x1b[22mBCD
	input := "\x1b[1mA\x1b[22mBCD"
	got := TruncateVisible(input, 2)
	// Should show 2 visible chars (A, B) and preserve ANSI codes.
	if !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Errorf("TruncateVisible ANSI = %q, should contain A and B", got)
	}
	if strings.Contains(got, "C") {
		t.Errorf("TruncateVisible ANSI = %q, should not contain C", got)
	}
	// Should end with reset.
	if !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("TruncateVisible ANSI = %q, should end with reset", got)
	}
}

func TestTruncateVisibleEmptyString(t *testing.T) {
	got := TruncateVisible("", 10)
	if got != "" {
		t.Errorf("TruncateVisible empty = %q, want empty", got)
	}
}

func TestTruncateVisibleANSIOnly(t *testing.T) {
	// Only ANSI codes, no visible characters.
	input := "\x1b[1m\x1b[0m"
	got := TruncateVisible(input, 5)
	if got != input {
		t.Errorf("TruncateVisible ANSI-only = %q, want %q", got, input)
	}
}
