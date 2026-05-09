// Package parse classifies lines of a .gitignore file into blank, comment, or entry.
package parse

import "strings"

type Kind int

const (
	KindBlank Kind = iota
	KindComment
	KindEntry
)

type Line struct {
	LineNo int
	Kind   Kind
	Text   string
}

// Parse splits content into lines and classifies each. Trailing CR is stripped
// from Text so callers see canonical line content; Parse does not preserve
// line endings (rewrite.Apply handles that on the original byte stream).
func Parse(content string) []Line {
	if len(content) == 0 {
		return nil
	}
	parts := strings.Split(content, "\n")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	lines := make([]Line, len(parts))
	for i, p := range parts {
		text := strings.TrimSuffix(p, "\r")
		lines[i] = Line{
			LineNo: i + 1,
			Kind:   classify(text),
			Text:   text,
		}
	}
	return lines
}

func classify(text string) Kind {
	if strings.TrimSpace(text) == "" {
		return KindBlank
	}
	if strings.HasPrefix(text, "#") {
		return KindComment
	}
	return KindEntry
}
