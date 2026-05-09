// Package testutil holds helpers shared by tests that need a real git repo
// in a temp directory. It is import-only from *_test.go files; production
// code does not depend on it.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// SetupRepo creates an isolated git repo in t.TempDir() with deterministic
// identity and gpgsign disabled, returning the absolute path.
func SetupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	RunGit(t, dir, "init", "-q", "-b", "main")
	RunGit(t, dir, "config", "user.email", "test@example.com")
	RunGit(t, dir, "config", "user.name", "Test")
	RunGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

// RunGit runs `git -C dir <args>` and fatals on non-zero exit.
func RunGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

// WriteFile writes content to path, creating parent directories as needed.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
