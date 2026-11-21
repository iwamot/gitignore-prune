package rewrite

import (
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
)

func TestApplyPreservesUnselectedBytes(t *testing.T) {
	texts := []string{"", "a", "*.log", "# comment", "   ", "\\#literal", "!*.log", "\tspaced  ", "日本語", "nul\x00byte", "\r"}
	property := func(choices []uint8, finalNewline bool) bool {
		var fragments, retained []string
		drop := map[int]bool{-1: true, 0: true, len(choices) + 1: true}
		all := make(map[int]bool)
		for index, choice := range choices {
			ending := "\n"
			if choice&2 != 0 {
				ending = "\r\n"
			}
			if index == len(choices)-1 && !finalNewline {
				ending = ""
			}
			fragment := texts[int(choice)%len(texts)] + ending
			fragments = append(fragments, fragment)
			deleted := choice&4 != 0
			drop[index+1] = deleted
			all[index+1] = true
			if !deleted {
				retained = append(retained, fragment)
			}
		}
		// The oracle concatenates input fragments selected before any bytes
		// are split or scanned by Apply, including explicit false map values.
		content := strings.Join(fragments, "")
		want := strings.Join(retained, "")
		if got := Apply(content, drop); got != want {
			t.Logf("content=%q drops=%v: got %q, want %q", content, drop, got, want)
			return false
		}
		if got := Apply(content, nil); got != content {
			t.Logf("no removals changed %q to %q", content, got)
			return false
		}
		if got := Apply(content, all); got != "" {
			t.Logf("removing every line from %q left %q", content, got)
			return false
		}
		return true
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 512, Rand: rand.New(rand.NewSource(1))}); err != nil {
		t.Fatal(err)
	}
}
