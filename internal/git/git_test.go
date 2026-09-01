package git

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

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
	dir := t.TempDir()
	if _, err := RepoRoot(dir); err == nil {
		t.Error("expected error for non-repo, got nil")
	}
}

func TestListGitignores(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "*.log\n")
	testutil.WriteFile(t, filepath.Join(repo, "subdir", ".gitignore"), "tmp/\n")
	testutil.WriteFile(t, filepath.Join(repo, "subdir", "README.md"), "")
	testutil.WriteFile(t, filepath.Join(repo, "untracked", ".gitignore"), "x\n")
	testutil.RunGit(t, repo, "add", ".gitignore", "subdir/.gitignore", "subdir/README.md")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	paths, err := ListGitignores(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".gitignore", "subdir/.gitignore"}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("ListGitignores = %v, want %v", paths, want)
	}
}

func TestListGitignores_emptyRepo(t *testing.T) {
	repo := testutil.SetupRepo(t)
	paths, err := ListGitignores(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Errorf("ListGitignores = %v, want empty", paths)
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
