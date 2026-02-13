package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Highlighter applies syntax highlighting to a single display line.
type Highlighter interface {
	Highlight(line string) string
}

// PlainHighlighter returns text unchanged.
type PlainHighlighter struct{}

func (PlainHighlighter) Highlight(line string) string { return line }

// MarkdownHighlighter applies ANSI colour codes to markdown syntax.
type MarkdownHighlighter struct{}

var (
	// Line-level patterns.
	reHeading = regexp.MustCompile(`^#{1,6}\s`)
	reQuote   = regexp.MustCompile(`^>\s`)
	reHR      = regexp.MustCompile(`^(---+|\*\*\*+|___+)\s*$`)

	// Inline patterns.
	reBold       = regexp.MustCompile(`(\*\*|__)(.+?)(\*\*|__)`)
	reItalic     = regexp.MustCompile(`(?:^|[^*_])(\*([^*]+?)\*|(?:^|\s)_([^_]+?)_)`)
	reCode       = regexp.MustCompile("`([^`]+?)`")
	reLink       = regexp.MustCompile(`\[([^\]]+?)\]\([^\)]+?\)`)
	reItalicStar = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*`)
	reItalicUs   = regexp.MustCompile(`(?:^|\s)_([^_]+?)_`)
)

func (MarkdownHighlighter) Highlight(line string) string {
	// Line-level rules: if matched, style the entire line.
	if reHR.MatchString(line) {
		return "\x1b[90m" + line + "\x1b[0m"
	}
	if reHeading.MatchString(line) {
		return "\x1b[1;34m" + line + "\x1b[0m"
	}
	if reQuote.MatchString(line) {
		return "\x1b[90m" + line + "\x1b[0m"
	}

	// Inline rules applied in order: bold, italic, code, link.
	result := line

	// Bold: **text** or __text__
	result = reBold.ReplaceAllString(result, "$1\x1b[1;33m$2\x1b[22;39m$3")

	// Italic *text* (not inside bold's **)
	result = reItalicStar.ReplaceAllStringFunc(result, func(match string) string {
		// The match may start with a non-* char; find the actual *...*
		idx := strings.Index(match, "*")
		prefix := match[:idx]
		inner := match[idx+1 : len(match)-1]
		return prefix + "*\x1b[3;36m" + inner + "\x1b[23;39m*"
	})

	// Italic _text_ (not inside a word)
	result = reItalicUs.ReplaceAllStringFunc(result, func(match string) string {
		idx := strings.Index(match, "_")
		prefix := match[:idx]
		inner := match[idx+1 : len(match)-1]
		return prefix + "_\x1b[3;36m" + inner + "\x1b[23;39m_"
	})

	// Inline code: `code`
	result = reCode.ReplaceAllString(result, "`\x1b[35m$1\x1b[39m`")

	// Links: [text](url) — underline the link text
	result = reLink.ReplaceAllStringFunc(result, func(match string) string {
		// Extract text between [ and ]
		open := strings.Index(match, "[")
		close := strings.Index(match, "]")
		if open < 0 || close < 0 {
			return match
		}
		text := match[open+1 : close]
		rest := match[close:]
		return "[" + "\x1b[4;32m" + text + "\x1b[24;39m" + rest
	})

	return result + "\x1b[0m"
}

// DetectHighlighter returns the appropriate highlighter for the given filename.
func DetectHighlighter(filename string) Highlighter {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md", ".markdown", ".mdx":
		return MarkdownHighlighter{}
	default:
		return PlainHighlighter{}
	}
}

// TruncateVisible truncates s to maxVisible visible characters,
// preserving ANSI escape sequences and appending a reset.
func TruncateVisible(s string, maxVisible int) string {
	var b strings.Builder
	visible := 0
	runes := []rune(s)
	i := 0

	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Consume the entire ANSI escape sequence.
			start := i
			i += 2 // skip \x1b[
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				i++
			}
			if i < len(runes) {
				i++ // skip the terminator
			}
			b.WriteString(string(runes[start:i]))
		} else {
			if visible >= maxVisible {
				break
			}
			b.WriteRune(runes[i])
			visible++
			i++
		}
	}

	// If we truncated, append reset to close any open ANSI spans.
	if visible >= maxVisible {
		b.WriteString("\x1b[0m")
	}

	return b.String()
}

// isAnsiTerminator returns true for the byte that ends a CSI sequence.
func isAnsiTerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
