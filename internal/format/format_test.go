package format

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
)

func TestReportsPreserveEntriesAndSummarizePruning(t *testing.T) {
	words := map[bool]string{true: "entry", false: "entries"}
	// The label column is eight characters wide, so entries line up whatever
	// their status.
	labels := map[Status]string{Keep: "[KEEP]  ", Prune: "[PRUNE] "}
	patterns := []string{"*.log", "!*.tmp", `\#literal`, " a b/ ", "日本語"}
	property := func(groups [][]uint8) bool {
		files := make([]FileResult, len(groups))
		sections := make([]string, len(groups))
		prunable := 0
		for i, group := range groups {
			files[i].Path = fmt.Sprintf("dir%d/日本語/.gitignore", i)
			section := fmt.Sprintf("=== %s (%d %s) ===\n", files[i].Path, len(group), words[len(group) == 1])
			for j, choice := range group {
				status := Keep
				if choice&1 != 0 {
					status = Prune
					prunable++
				}
				pattern := patterns[int(choice)%len(patterns)]
				text := fmt.Sprintf("%s<%d:%d>%s", pattern, i, j, pattern)
				files[i].Entries = append(files[i].Entries, Entry{Status: status, Text: text})
				section += labels[status] + text + "\n"
			}
			sections[i] = section
		}
		// A blank line separates the sections from each other and from the
		// summary.
		body := strings.Join(append(sections, ""), "\n")
		checkSummary, fixSummary := "No prunable entries found.\n", "No prunable entries found.\n"
		if prunable > 0 {
			checkSummary = "Run with --fix to prune entries marked [PRUNE].\n"
			fixSummary = fmt.Sprintf("Pruned %d %s.\n", prunable, words[prunable == 1])
		}
		check, fix := Check(files), Fix(files)
		if check != body+checkSummary || fix != body+fixSummary {
			t.Logf("groups=%v\nCheck:\n%s\nwant:\n%s\nFix:\n%s\nwant:\n%s", groups, check, body+checkSummary, fix, body+fixSummary)
			return false
		}
		return true
	}
	// Guarantee the empty, singular, plural, and multi-file boundaries as well as generated inputs.
	for _, groups := range [][][]uint8{nil, {nil}, {{0}}, {{1}}, {{1, 1, 0}}, {{1}, {0}}} {
		if !property(groups) {
			t.Fatalf("report contract failed for %v", groups)
		}
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 256, Rand: rand.New(rand.NewSource(1))}); err != nil {
		t.Fatal(err)
	}
}
