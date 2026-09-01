// Package pattern rewrites .gitignore patterns so that git names the same
// paths when it reads them from the command line instead of the .gitignore.
package pattern

import "strings"

// Reanchor returns p rewritten so that, read by git from an --exclude-from
// file, it names the same paths it names inside the .gitignore at dir. dir is
// the .gitignore's directory relative to the repository root, slash-separated,
// "." for the root.
//
// Git resolves --exclude-from patterns against the repository root, whereas a
// .gitignore resolves them against its own directory. The two agree for
// patterns without a directory separator (they match at any depth), so those
// are returned unchanged; a separator at the beginning or in the middle
// anchors the pattern, and dir is prepended so that it stays pointed at the
// same subtree.
func Reanchor(dir, p string) string {
	if dir == "." || !anchored(p) {
		return p
	}
	return "/" + dir + "/" + strings.TrimPrefix(p, "/")
}

// anchored reports whether git treats p as relative to its .gitignore's
// directory: once trailing spaces are trimmed the way git trims them, a
// separator anywhere but the very end makes it so.
func anchored(p string) bool {
	p = trimTrailingSpaces(p)
	return strings.Contains(strings.TrimSuffix(p, "/"), "/")
}

// trimTrailingSpaces mirrors git's trim_trailing_spaces (dir.c): unescaped
// trailing spaces are dropped, and a backslash protects the character after
// it. A backslash at the very end leaves the pattern as it is.
func trimTrailingSpaces(p string) string {
	lastSpace := -1
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case ' ':
			if lastSpace < 0 {
				lastSpace = i
			}
		case '\\':
			i++
			if i == len(p) {
				return p
			}
			lastSpace = -1
		default:
			lastSpace = -1
		}
	}
	if lastSpace < 0 {
		return p
	}
	return p[:lastSpace]
}
