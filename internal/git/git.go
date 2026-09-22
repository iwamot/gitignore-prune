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
	"slices"
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

// ListGitignores returns repo-root-relative paths, in path order, of every
// tracked or untracked file whose basename is ".gitignore". Untracked ones
// are included because a .gitignore copied from a template is usually
// still unstaged the first time the tool runs. --exclude-standard keeps the
// untracked walk out of ignored directories, so a vendored
// node_modules/foo/.gitignore is not reported. Submodule and nested
// repository .gitignores are excluded automatically: their files live in
// the inner index, and the untracked walk lists an inner repository as a
// single directory without entering it. The listing is read NUL-separated
// because the line-based form quotes paths with non-ASCII or special
// bytes, and sorted because git prints untracked paths before tracked ones.
func ListGitignores(repoRoot string) ([]string, error) {
	out, err := exec.Command("git", "-C", repoRoot, "ls-files", "-z", "-c", "-o", "--exclude-standard").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var paths []string
	for line := range strings.SplitSeq(string(out), "\x00") {
		if line == "" {
			continue
		}
		if filepath.Base(line) == ".gitignore" {
			paths = append(paths, line)
		}
	}
	slices.Sort(paths)
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
