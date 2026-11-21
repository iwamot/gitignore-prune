package format

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"testing/quick"
)

func TestReportsPreserveEntriesAndSummarizePruning(t *testing.T) {
	words := map[bool]string{true: "entry", false: "entries"}
	labels := map[Status]string{Keep: "[KEEP]", Prune: "[PRUNE]"}
	patterns := []string{"*.log", "!*.tmp", `\#literal`, " a b/ ", "日本語"}
	splitReport := func(report string) ([]string, string, bool) {
		lines := strings.Split(report, "\n")
		if len(lines) < 2 || lines[len(lines)-1] != "" {
			return nil, "", false
		}
		body := slices.DeleteFunc(lines[:len(lines)-2], func(line string) bool { return line == "" })
		return body, lines[len(lines)-2], true
	}
	property := func(groups [][]uint8) bool {
		files := make([]FileResult, len(groups))
		prunable := 0
		for i, group := range groups {
			files[i].Path = fmt.Sprintf("dir%d/日本語/.gitignore", i)
			for j, choice := range group {
				status := Keep
				if choice&1 != 0 {
					status = Prune
					prunable++
				}
				pattern := patterns[int(choice)%len(patterns)]
				files[i].Entries = append(files[i].Entries, Entry{
					Status: status,
					Text:   fmt.Sprintf("%s<%d:%d>%s", pattern, i, j, pattern),
				})
			}
		}
		// Preserve the input order independently of any changes made by rendering.
		wantFiles := slices.Clone(files)
		for i := range wantFiles {
			wantFiles[i].Entries = slices.Clone(files[i].Entries)
		}
		checkBody, checkSummary, checkOK := splitReport(Check(files))
		fixBody, fixSummary, fixOK := splitReport(Fix(files))
		if !checkOK || !fixOK || !slices.Equal(checkBody, fixBody) {
			return false
		}
		row := 0
		for _, file := range wantFiles {
			count := len(file.Entries)
			if row >= len(checkBody) || !strings.HasPrefix(checkBody[row], "=== "+file.Path+" ") ||
				!strings.HasSuffix(checkBody[row], fmt.Sprintf("(%d %s) ===", count, words[count == 1])) {
				return false
			}
			row++
			for _, entry := range file.Entries {
				if row >= len(checkBody) || !strings.HasSuffix(checkBody[row], entry.Text) ||
					strings.TrimSpace(strings.TrimSuffix(checkBody[row], entry.Text)) != labels[entry.Status] {
					return false
				}
				row++
			}
		}
		if row != len(checkBody) {
			return false
		}
		if prunable == 0 {
			return checkSummary == "No prunable entries found." && fixSummary == checkSummary
		}
		return checkSummary == "Run with --fix to prune entries marked [PRUNE]." &&
			fixSummary == fmt.Sprintf("Pruned %d %s.", prunable, words[prunable == 1])
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
