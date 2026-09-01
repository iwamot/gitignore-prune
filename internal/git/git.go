// Package git wraps the git CLI calls used by gitignore-prune. The package
// is the I/O boundary; matching is delegated to git so that interpretation
// of .gitignore patterns is identical to git's own behavior.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/iwamot/gitignore-prune/internal/pattern"
)

// RepoRoot returns the absolute path of the working tree root containing path.
func RepoRoot(path string) (string, error) {
	out, err := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ListGitignores returns repo-root-relative paths of every tracked file whose
// basename is ".gitignore". Submodule .gitignores are excluded automatically:
// they live in the submodule's index, not the parent's, so git ls-files at
// the parent never sees them.
func ListGitignores(repoRoot string) ([]string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "ls-files").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		if filepath.Base(line) == ".gitignore" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

// ShouldPrune reports whether entryText, an entry of the .gitignore at
// gitignorePath (relative to repoRoot, as returned by ListGitignores), fails
// to match any path in the working tree. The pattern is fed to git via a
// one-line --exclude-from temp file, and both tracked (-c) and untracked
// (-o) ignored listings are checked; a hit in either marks the entry as
// matching, so it stays. A leading "!" is stripped before matching, since
// negation does not change which paths the pattern names.
//
// --exclude-from knows nothing about where the .gitignore lives, so its
// scope is restored in two parts: git runs in the .gitignore's directory so
// that only paths below it are listed, and the pattern is reanchored to the
// repository root (see pattern.Reanchor) because git resolves --exclude-from
// patterns from there rather than from the current directory.
func ShouldPrune(repoRoot, gitignorePath, entryText string) (bool, error) {
	dir := path.Dir(gitignorePath)
	probe := pattern.Reanchor(dir, strings.TrimPrefix(entryText, "!"))

	tmp, err := os.CreateTemp("", "gitignore-prune-*.txt")
	if err != nil {
		return false, fmt.Errorf("create temp: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(probe + "\n"); err != nil {
		tmp.Close()
		return false, fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close temp: %w", err)
	}

	for _, mode := range []string{"-c", "-o"} {
		out, err := exec.Command("git", "-C", filepath.Join(repoRoot, filepath.FromSlash(dir)),
			"ls-files", mode, "-i", "--exclude-from="+tmp.Name()).Output()
		if err != nil {
			return false, fmt.Errorf("git ls-files %s: %w", mode, err)
		}
		if len(bytes.TrimSpace(out)) > 0 {
			return false, nil
		}
	}
	return true, nil
}
