// Package testutil holds helpers shared by tests that need a real git repo
// in a temp directory. It is import-only from *_test.go files; production
// code does not depend on it.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// SetupRepo creates an isolated git repo in t.TempDir() with deterministic
// identity and gpgsign disabled, returning the absolute path.
func SetupRepo(t *testing.T) string {
	t.Helper()
	isolateGitEnv(t)
	dir := t.TempDir()
	RunGit(t, dir, "init", "-q", "-b", "main")
	RunGit(t, dir, "config", "user.email", "test@example.com")
	RunGit(t, dir, "config", "user.name", "Test")
	RunGit(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

// isolateGitEnv drops every GIT_* variable from the process environment for
// the rest of the test. Git hands a hook its own git environment, and
// `git commit -- <path>` makes that GIT_INDEX_FILE an absolute path to a
// temporary index of the repository being committed. Both the helpers here
// and the code under test spawn git, so an inherited value sends those
// commands at the outer repository instead of the one in t.TempDir().
func isolateGitEnv(t *testing.T) {
	t.Helper()
	for _, name := range gitEnvNames(os.Environ()) {
		value := os.Getenv(name)
		t.Cleanup(func() { _ = os.Setenv(name, value) })
		_ = os.Unsetenv(name)
	}
}

// gitEnvNames returns the names of the GIT_* entries in environ.
func gitEnvNames(environ []string) []string {
	var names []string
	for _, entry := range environ {
		name, _, found := strings.Cut(entry, "=")
		if found && strings.HasPrefix(name, "GIT_") {
			names = append(names, name)
		}
	}
	return names
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
