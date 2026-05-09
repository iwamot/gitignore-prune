// Package rewrite generates the new .gitignore byte stream after dropping
// selected line numbers, preserving the original line endings (LF / CRLF /
// mixed) and the rest of the bytes verbatim.
package rewrite

import "strings"

// Apply walks content byte-by-byte, splitting on '\n'. Lines whose 1-indexed
// number is in dropLines are skipped; everything else (including comments,
// blank lines, surrounding whitespace, and original \r\n sequences) is copied
// verbatim. A trailing partial line (no final '\n') is supported.
func Apply(content string, dropLines map[int]bool) string {
	if len(content) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(len(content))
	lineNo := 1
	start := 0
	for i := 0; i < len(content); i++ {
		if content[i] != '\n' {
			continue
		}
		if !dropLines[lineNo] {
			sb.WriteString(content[start : i+1])
		}
		lineNo++
		start = i + 1
	}
	if start < len(content) && !dropLines[lineNo] {
		sb.WriteString(content[start:])
	}
	return sb.String()
}
