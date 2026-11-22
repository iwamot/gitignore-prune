package pattern_test

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	gitio "github.com/iwamot/gitignore-prune/internal/git"
	"github.com/iwamot/gitignore-prune/internal/testutil"
)

func TestReanchorPreservesGitMatching(t *testing.T) {
	repo := testutil.SetupRepo(t)
	files := []string{"foo", "app.log", "bar/baz", "build/root", "tree/root", "a b/c", "#literal", "!literal"}
	// Git's index can represent trailing-space names without checking them
	// out on filesystems that cannot create them.
	indexOnly := []string{"build/ "}
	// Directory names that Git would read as glob syntax, a comment or a
	// negation if the directory were written into a pattern as it is.
	nested := []string{"sub", "a/b", "[ab]", `a\b`, "a?c", "s*", "#x", "!x"}
	for _, dir := range nested {
		for _, name := range []string{"deep/foo", "deep/app.log", "bar/baz", "deep/build/item", "tree/child", "a b/c", "deep/cache/item"} {
			files = append(files, dir+"/"+name)
		}
		indexOnly = append(indexOnly, dir+"/build/ ")
	}
	for _, name := range files {
		testutil.WriteFile(t, filepath.Join(repo, filepath.FromSlash(name)), "")
	}
	emptyBlob := strings.TrimSpace(testutil.RunGit(t, repo, "hash-object", "-w", "--stdin"))
	patterns := []string{
		"foo", "/foo", "bar/baz", "/bar/baz", "*.log", "cache/", "/tree/", "/tree", "**/foo", "**/baz", "tree/**",
		"cache/  ", "build/\\ ", "build/\\", "a b/c", "\\#literal", "\\!literal", "missing",
	}
	dirs := append([]string{".", "empty"}, nested...)
	previous := ""
	property := func(p, dir string, negated, tracked bool) bool {
		ignore := path.Join(dir, ".gitignore")
		if previous != "" {
			if err := os.Remove(filepath.Join(repo, filepath.FromSlash(previous))); err != nil {
				t.Fatal(err)
			}
		}
		testutil.WriteFile(t, filepath.Join(repo, filepath.FromSlash(ignore)), p+"\n")
		previous = ignore
		testutil.RunGit(t, repo, "read-tree", "--empty")
		if tracked {
			testutil.RunGit(t, repo, "add", "-f", "--", ".")
		}
		addIndexOnly := []string{"update-index", "--add"}
		for _, name := range indexOnly {
			addIndexOnly = append(addIndexOnly, "--cacheinfo", "100644,"+emptyBlob+","+name)
		}
		testutil.RunGit(t, repo, addIndexOnly...)

		// The oracle reads the original pattern from its actual .gitignore.
		// --no-index includes tracked paths; the prune API treats a negation
		// as naming the same paths as its positive counterpart.
		candidates := append(append(append([]string{}, files...), indexOnly...), ignore)
		cmd := exec.Command("git", "-C", repo, "-c", "core.excludesFile="+os.DevNull,
			"check-ignore", "--no-index", "-z", "--stdin")
		cmd.Stdin = strings.NewReader(strings.Join(candidates, "\x00") + "\x00")
		matches, err := cmd.Output()
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 {
				t.Fatalf("git check-ignore: %v", err)
			}
		}
		entry := p
		if negated {
			entry = "!" + p
		}
		got, err := gitio.ShouldPrune(repo, ignore, entry)
		if err != nil {
			t.Fatal(err)
		}
		want := len(matches) == 0
		if got != want {
			t.Logf("dir=%q pattern=%q negated=%t tracked=%t: prune=%t, Git matches=%q", dir, p, negated, tracked, got, matches)
		}
		return got == want
	}
	// The fixture domains are finite, so check every combination against Git.
	for _, p := range patterns {
		for _, dir := range dirs {
			for _, negated := range []bool{false, true} {
				for _, tracked := range []bool{false, true} {
					if !property(p, dir, negated, tracked) {
						t.FailNow()
					}
				}
			}
		}
	}
}
