package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"
	"testing/quick"

	"github.com/iwamot/gitignore-prune/internal/testutil"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		argv    []string
		want    cliArgs
		wantErr bool
	}{
		{"empty", nil, cliArgs{path: "."}, false},
		{"path only", []string{"some/dir"}, cliArgs{path: "some/dir"}, false},
		{"fix only", []string{"--fix"}, cliArgs{fix: true, path: "."}, false},
		{"fix then path", []string{"--fix", "p"}, cliArgs{fix: true, path: "p"}, false},
		{"path then fix", []string{"p", "--fix"}, cliArgs{fix: true, path: "p"}, false},
		{"help short", []string{"-h"}, cliArgs{showHelp: true, path: "."}, false},
		{"help long", []string{"--help"}, cliArgs{showHelp: true, path: "."}, false},
		{"version short", []string{"-v"}, cliArgs{showVersion: true, path: "."}, false},
		{"version long", []string{"--version"}, cliArgs{showVersion: true, path: "."}, false},
		{"unknown flag", []string{"--unknown"}, cliArgs{}, true},
		{"two paths", []string{"a", "b"}, cliArgs{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.argv)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseArgs err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseArgs = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveVersionHonorsSourcePriority(t *testing.T) {
	property := func(major, minor, patch uint16) bool {
		injected := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		// Interior v characters belong to the version, even when a leading v is removed.
		stable := fmt.Sprintf("%d.%d.%d", patch, major, minor)
		recorded := fmt.Sprintf("%s-preview.v%d+build.v%d", stable, minor, major)
		sources := []struct {
			info *debug.BuildInfo
			want string
		}{
			{nil, devVersion},
			{&debug.BuildInfo{}, devVersion},
			{&debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, devVersion},
			{&debug.BuildInfo{Main: debug.Module{Version: stable}}, stable},
			{&debug.BuildInfo{Main: debug.Module{Version: "v" + stable}}, stable},
			{&debug.BuildInfo{Main: debug.Module{Version: recorded}}, recorded},
			{&debug.BuildInfo{Main: debug.Module{Version: "v" + recorded}}, recorded},
		}
		for _, source := range sources {
			gotInjected := resolveVersion(injected, source.info)
			gotFallback := resolveVersion(devVersion, source.info)
			if gotInjected != injected || gotFallback != source.want {
				t.Logf("info=%+v: injected=%q, want %q; fallback=%q, want %q", source.info, gotInjected, injected, gotFallback, source.want)
				return false
			}
		}
		return true
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 256, Rand: rand.New(rand.NewSource(1))}); err != nil {
		t.Fatal(err)
	}
}

func TestRunPrunesOnlyUnmatchedEntriesAndIsIdempotent(t *testing.T) {
	hasEntry := func(report, text, label string) bool {
		for line := range strings.SplitSeq(report, "\n") {
			if strings.HasSuffix(line, text) && strings.TrimSpace(strings.TrimSuffix(line, text)) == label {
				return true
			}
		}
		return false
	}
	property := func(choices []uint8, crlf bool) bool {
		repo := testutil.SetupRepo(t)
		ending := "\n"
		if crlf {
			ending = "\r\n"
		}
		files := []struct{ path, original, want string }{
			{path: ".gitignore"}, {path: "subdir/.gitignore"}, {path: "a/b/.gitignore"},
		}
		for i := range files {
			files[i].original = "# 日本語 comment" + ending + ending
			files[i].want = files[i].original
		}
		entries := map[string]string{}
		var tracked []string
		prunable := 0
		for i, choice := range choices[:min(len(choices), 12)] {
			file := &files[i%len(files)]
			dir := filepath.Dir(file.path)
			stem := fmt.Sprintf("item-%02d", i)
			forms := []struct{ pattern, target string }{
				{stem + "*.log", stem + "-app.log"}, {"/" + stem, stem + "/x"},
				{"lib/" + stem, "lib/" + stem}, {"cache-" + stem + "/", "cache-" + stem + "/x"},
			}
			form := forms[int(choice>>2)%len(forms)]
			text := form.pattern
			if choice&16 != 0 {
				text = "!" + text
			}
			file.original += text + ending
			entries[text] = "[PRUNE]"
			if choice&1 != 0 {
				file.want += text + ending
				entries[text] = "[KEEP]"
				rel := filepath.Join(dir, filepath.FromSlash(form.target))
				testutil.WriteFile(t, filepath.Join(repo, rel), "")
				if choice&2 != 0 {
					tracked = append(tracked, rel)
				}
			} else {
				prunable++
			}
		}
		for i := range files {
			// An unterminated comment must survive both newline styles unchanged.
			files[i].original += "# tail"
			files[i].want += "# tail"
			testutil.WriteFile(t, filepath.Join(repo, files[i].path), files[i].original)
		}
		if len(tracked) > 0 {
			testutil.RunGit(t, repo, append([]string{"add", "-f", "--"}, tracked...)...)
		}
		// Check is read-only; fix removes only tagged unmatched lines; a second fix is a no-op.
		for phase := range 3 {
			args, wantExit := []string{repo}, exitOK
			if phase > 0 {
				args = []string{"--fix", repo}
			} else if prunable > 0 {
				wantExit = exitPrunesFound
			}
			var out, errBuf bytes.Buffer
			if code := run(args, &out, &errBuf); code != wantExit || errBuf.Len() != 0 {
				t.Logf("choices=%v CRLF=%t phase=%d: exit=%d, want %d; stderr=%q", choices, crlf, phase, code, wantExit, errBuf.String())
				return false
			}
			for _, file := range files {
				want := file.want
				if phase == 0 {
					want = file.original
				}
				got, err := os.ReadFile(filepath.Join(repo, file.path))
				if err != nil || string(got) != want {
					t.Logf("phase=%d file=%q: content=%q, want %q; err=%v", phase, file.path, got, want, err)
					return false
				}
			}
			if phase < 2 {
				for text, label := range entries {
					if !hasEntry(out.String(), text, label) {
						t.Logf("phase=%d: missing %s %q in report %q", phase, label, text, out.String())
						return false
					}
				}
			}
			wantSummary := "No prunable entries found."
			if prunable > 0 && phase == 0 {
				wantSummary = "Run with --fix"
			} else if prunable > 0 && phase == 1 {
				wantSummary = fmt.Sprintf("Pruned %d ", prunable)
			}
			if !strings.Contains(out.String(), wantSummary) {
				t.Logf("phase=%d: missing summary %q in report %q", phase, wantSummary, out.String())
				return false
			}
		}
		return true
	}
	for _, choices := range [][]uint8{nil, {0}, {1}, {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}} {
		for _, crlf := range []bool{false, true} {
			if !property(choices, crlf) {
				t.Fatalf("pruning contract failed for %v, CRLF=%t", choices, crlf)
			}
		}
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 32, Rand: rand.New(rand.NewSource(1))}); err != nil {
		t.Fatal(err)
	}
}
