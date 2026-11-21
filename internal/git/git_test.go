package git

import (
	"fmt"
	"maps"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/quick"

	"github.com/iwamot/gitignore-prune/internal/testutil"
)

func TestRepoRoot_atRoot(t *testing.T) {
	repo := testutil.SetupRepo(t)
	got, err := RepoRoot(repo)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(repo)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Errorf("RepoRoot(%q) = %q, want %q", repo, got, want)
	}
}

func TestRepoRoot_fromSubdir(t *testing.T) {
	repo := testutil.SetupRepo(t)
	sub := filepath.Join(repo, "subdir")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := RepoRoot(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(repo)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Errorf("RepoRoot(%q) = %q, want %q", sub, got, want)
	}
}

func TestRepoRoot_notARepo(t *testing.T) {
	testutil.IsolateGitEnv(t)
	dir := t.TempDir()
	if _, err := RepoRoot(dir); err == nil {
		t.Error("expected error for non-repo, got nil")
	}
}

// Listing depends on visibility and index membership, not on how Git groups
// tracked and untracked paths in its output.
func TestListGitignoresPreservesVisibleInventory(t *testing.T) {
	emptyRepo := testutil.SetupRepo(t)
	if paths, err := ListGitignores(emptyRepo); err != nil || len(paths) != 0 {
		t.Fatalf("empty repository: paths = %q, err = %v", paths, err)
	}
	property := func(choices []uint8) bool {
		repo := testutil.SetupRepo(t)
		testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "/ignored/\n")
		want := map[string]bool{".gitignore": true}
		tracked := []string{}
		for i, choice := range choices {
			// The low bits choose independent tracked/ignored states; the rest vary paths.
			prefix := "visible"
			if choice&2 != 0 {
				prefix = "ignored"
			}
			names := []string{"plain", "日本語", "with spaces", "nested/deep", "[brackets]"}
			dir := fmt.Sprintf("%s/case-%02d/%s", prefix, i, names[int(choice>>2)%len(names)])
			for _, name := range []string{".gitignore", ".gitignore.bak", "prefix.gitignore"} {
				rel := dir + "/" + name
				testutil.WriteFile(t, filepath.Join(repo, filepath.FromSlash(rel)), "")
				if choice&1 != 0 {
					tracked = append(tracked, rel)
				}
			}
			if choice&1 != 0 || choice&2 == 0 {
				want[dir+"/.gitignore"] = true
			}
		}
		// An untracked .gitignore can also exclude itself.
		testutil.WriteFile(t, filepath.Join(repo, "self-ignored", ".gitignore"), ".gitignore\n")
		if len(tracked) > 0 {
			testutil.RunGit(t, repo, append([]string{"add", "-f", "--"}, tracked...)...)
		}
		checkInventory := func() bool {
			paths, err := ListGitignores(repo)
			if err != nil || len(paths) != len(want) || !slices.IsSorted(paths) {
				t.Logf("choices %v: paths = %q, want inventory %v, err = %v", choices, paths, want, err)
				return false
			}
			remaining := maps.Clone(want)
			for _, rel := range paths {
				if !remaining[rel] {
					t.Logf("choices %v: unexpected or duplicate path %q", choices, rel)
					return false
				}
				delete(remaining, rel)
			}
			return len(remaining) == 0
		}
		if !checkInventory() {
			return false
		}
		// Staging visible files must preserve their inclusion and ordering.
		visible := slices.Sorted(maps.Keys(want))
		testutil.RunGit(t, repo, append([]string{"add", "-f", "--"}, visible...)...)
		return checkInventory()
	}
	for _, choices := range [][]uint8{nil, {0}, {1}, {2}, {3}, {0, 1, 2, 3}} {
		if !property(choices) {
			t.Fatalf("listing contract failed for %v", choices)
		}
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 64, Rand: rand.New(rand.NewSource(1))}); err != nil {
		t.Fatal(err)
	}
}

func TestShouldPrune_untrackedMatches(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	got, err := ShouldPrune(repo, ".gitignore", "*.log")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("ShouldPrune(*.log) = prune, want keep (untracked file matches)")
	}
}

func TestShouldPrune_trackedMatches(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "main.go"), "")
	testutil.RunGit(t, repo, "add", "main.go")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")
	got, err := ShouldPrune(repo, ".gitignore", "*.go")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("ShouldPrune(*.go) = prune, want keep (tracked file matches)")
	}
}

func TestShouldPrune_noMatch(t *testing.T) {
	repo := testutil.SetupRepo(t)
	got, err := ShouldPrune(repo, ".gitignore", "node_modules/")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("ShouldPrune(node_modules/) = keep, want prune (nothing matches)")
	}
}

func TestShouldPrune_negationStripped(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	got, err := ShouldPrune(repo, ".gitignore", "!*.log")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("ShouldPrune(!*.log) = prune, want keep (negation pattern still names *.log which exists)")
	}
}

func TestShouldPrune_negationNoMatch(t *testing.T) {
	repo := testutil.SetupRepo(t)
	got, err := ShouldPrune(repo, ".gitignore", "!nonexistent/")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("ShouldPrune(!nonexistent/) = keep, want prune")
	}
}

func TestShouldPrune_subdirScope(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "subdir", "tmp", "x"), "")

	got, err := ShouldPrune(repo, "subdir/.gitignore", "tmp/")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("ShouldPrune from subdir for 'tmp/' = prune, want keep")
	}

	got, err = ShouldPrune(repo, "subdir/.gitignore", "nonexistent/")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("ShouldPrune from subdir for 'nonexistent/' = keep, want prune")
	}
}

func TestShouldPrune_errorOnNonRepo(t *testing.T) {
	testutil.IsolateGitEnv(t)
	dir := t.TempDir()
	if _, err := ShouldPrune(dir, ".gitignore", "*.log"); err == nil {
		t.Error("expected error when called outside a git repo")
	}
}

func TestShouldPrune_subdirAnchoredUntracked(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "subdir", "foo"), "")
	testutil.WriteFile(t, filepath.Join(repo, "subdir", "bar", "baz"), "")

	for _, entry := range []string{"/foo", "bar/baz", "/bar/baz", "**/baz"} {
		got, err := ShouldPrune(repo, "subdir/.gitignore", entry)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Errorf("ShouldPrune(subdir/.gitignore, %q) = prune, want keep (untracked file matches)", entry)
		}
	}
}

func TestShouldPrune_subdirAnchoredTracked(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "subdir", "foo"), "")
	testutil.RunGit(t, repo, "add", "subdir/foo")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	got, err := ShouldPrune(repo, "subdir/.gitignore", "/foo")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("ShouldPrune(subdir/.gitignore, /foo) = prune, want keep (tracked file matches)")
	}
}

func TestShouldPrune_subdirAnchoredStaysInSubtree(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "foo"), "")
	testutil.WriteFile(t, filepath.Join(repo, "subdir", "deep", "foo"), "")

	got, err := ShouldPrune(repo, "subdir/.gitignore", "/foo")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("ShouldPrune(subdir/.gitignore, /foo) = keep, want prune (only root foo and subdir/deep/foo exist)")
	}
}

func TestShouldPrune_nestedDirAnchored(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, "a", "b", "out", "x"), "")

	got, err := ShouldPrune(repo, "a/b/.gitignore", "/out")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("ShouldPrune(a/b/.gitignore, /out) = prune, want keep")
	}
}
