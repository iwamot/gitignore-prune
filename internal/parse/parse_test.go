package parse

import (
	"math/rand"
	"slices"
	"strings"
	"testing"
	"testing/quick"
)

func TestParsePreservesGeneratedLinesAndKinds(t *testing.T) {
	bodies := []struct {
		text string
		kind Kind
	}{
		{"", KindBlank}, {"   ", KindBlank}, {"\t", KindBlank}, {"\u2003", KindBlank},
		{"#", KindComment}, {"# comment", KindComment}, {"# 日本語", KindComment}, {"# inner\rvalue", KindComment},
		{"*.log", KindEntry}, {"foo/", KindEntry}, {"\\#foo", KindEntry}, {"!*.log", KindEntry},
		{" #literal", KindEntry}, {"日本語", KindEntry}, {"inner\rvalue", KindEntry}, {"name  ", KindEntry},
	}
	endings := []struct{ raw, suffix string }{{"\n", ""}, {"\r\n", ""}, {"\r\r\n", "\r"}}
	property := func(choices []uint8, finalNewline bool) bool {
		var fragments []string
		var want []Line
		for index, choice := range choices {
			body := bodies[int(choice)%len(bodies)]
			ending := endings[int(choice>>4)%len(endings)]
			if index == len(choices)-1 && !finalNewline {
				ending.raw, ending.suffix = "", ""
				if choice&128 != 0 {
					ending.raw = "\r"
				}
			}
			fragment := body.text + ending.raw
			if fragment == "" {
				continue // An empty final fragment is not an additional physical line.
			}
			fragments = append(fragments, fragment)
			want = append(want, Line{LineNo: len(want) + 1, Kind: body.kind, Text: body.text + ending.suffix})
		}
		// Kinds and line boundaries are known when inputs are generated;
		// the oracle does not split or classify the assembled document.
		content := strings.Join(fragments, "")
		if got := Parse(content); !slices.Equal(got, want) {
			t.Logf("Parse(%q) = %#v, want %#v", content, got, want)
			return false
		}
		return true
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 512, Rand: rand.New(rand.NewSource(1))}); err != nil {
		t.Fatal(err)
	}
}
