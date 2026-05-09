// Package format produces the human-readable stdout report.
package format

import (
	"fmt"
	"strings"
)

type Status int

const (
	Keep Status = iota
	Prune
)

type Entry struct {
	Status Status
	Text   string
}

type FileResult struct {
	Path    string
	Entries []Entry
}

// Check renders the default-mode report: one section per file followed by a
// summary that reflects whether any entry needs pruning.
func Check(files []FileResult) string {
	return render(files, false)
}

// Fix renders the same per-file body as Check, followed by a summary that
// reflects what was (or wasn't) actually removed.
func Fix(files []FileResult) string {
	return render(files, true)
}

func render(files []FileResult, fixed bool) string {
	var sb strings.Builder
	pruneCount := 0
	for i, f := range files {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "=== %s (%d %s) ===\n", f.Path, len(f.Entries), entryWord(len(f.Entries)))
		for _, e := range f.Entries {
			label := "[KEEP]"
			if e.Status == Prune {
				label = "[PRUNE]"
				pruneCount++
			}
			fmt.Fprintf(&sb, "%-8s%s\n", label, e.Text)
		}
	}
	if len(files) > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString(summary(pruneCount, fixed))
	return sb.String()
}

func summary(pruneCount int, fixed bool) string {
	if fixed && pruneCount > 0 {
		return fmt.Sprintf("Pruned %d %s.\n", pruneCount, entryWord(pruneCount))
	}
	if pruneCount > 0 {
		return "Run with --fix to prune entries marked [PRUNE].\n"
	}
	return "No prunable entries found.\n"
}

func entryWord(n int) string {
	if n == 1 {
		return "entry"
	}
	return "entries"
}
