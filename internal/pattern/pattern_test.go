package pattern_test

import (
	"errors"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"

	gitio "github.com/iwamot/gitignore-prune/internal/git"
	"github.com/iwamot/gitignore-prune/internal/testutil"
)

func TestReanchorPreservesGitMatching(t *testing.T) {
	repo := testutil.SetupRepo(t)
	files := []string{
		"foo", "app.log", "bar/baz", "build/root", "tree/root", "a b/c", "#literal", "!literal",
		"sub/deep/foo", "sub/deep/app.log", "sub/bar/baz", "sub/deep/build/item", "sub/tree/child", "sub/a b/c", "sub/deep/cache/item",
		"a/b/foo", "a/b/deep/app.log", "a/b/bar/baz", "a/b/deep/build/item", "a/b/tree/child", "a/b/a b/c", "a/b/deep/cache/item",
	}
	for _, name := range files {
		testutil.WriteFile(t, filepath.Join(repo, filepath.FromSlash(name)), "")
	}
	// Git's index can represent trailing-space names without checking them
	// out on filesystems that cannot create them.
	indexOnly := []string{"build/ ", "sub/build/ ", "a/b/build/ "}
	emptyBlob := strings.TrimSpace(testutil.RunGit(t, repo, "hash-object", "-w", "--stdin"))
	patterns := []string{
		"foo", "/foo", "bar/baz", "*.log", "cache/", "/tree/", "**/foo", "tree/**",
		"cache/  ", "build/\\ ", "build/\\", "a b/c", "\\#literal", "\\!literal", "missing",
	}
	dirs := []string{".", "sub", "a/b", "empty"}
	previous := ""
	property := func(patternChoice, dirChoice uint8, negated, tracked bool) bool {
		p := patterns[int(patternChoice)%len(patterns)]
		dir := dirs[int(dirChoice)%len(dirs)]
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
		for _, name := range indexOnly {
			testutil.RunGit(t, repo, "update-index", "--add", "--cacheinfo", "100644,"+emptyBlob+","+name)
		}

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
	if err := quick.Check(property, &quick.Config{MaxCount: 256, Rand: rand.New(rand.NewSource(1))}); err != nil {
		t.Fatal(err)
	}
}
