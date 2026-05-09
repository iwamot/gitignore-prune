package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/iwamot/gitignore-prune/internal/format"
	"github.com/iwamot/gitignore-prune/internal/git"
	"github.com/iwamot/gitignore-prune/internal/parse"
	"github.com/iwamot/gitignore-prune/internal/rewrite"
)

const (
	exitOK          = 0
	exitPrunesFound = 1
	exitUsage       = 2
)

var version = "0.0.0-dev"

const helpText = `gitignore-prune — remove .gitignore entries that match nothing in the working tree.

Usage:
  gitignore-prune [<path>]    inspect .gitignore files in the repo containing <path> (default: cwd)
  gitignore-prune --fix       remove unmatched entries in place
  gitignore-prune -h, --help
  gitignore-prune -V, --version

Exit codes:
  0  no entries to prune, or --fix succeeded
  1  unmatched entries found (default mode only)
  2  usage error, or path is not inside a git repository
`

type cliArgs struct {
	showHelp    bool
	showVersion bool
	fix         bool
	path        string
}

func parseArgs(argv []string) (cliArgs, error) {
	a := cliArgs{path: "."}
	pathSet := false
	for _, arg := range argv {
		switch arg {
		case "-h", "--help":
			a.showHelp = true
		case "-V", "--version":
			a.showVersion = true
		case "--fix":
			a.fix = true
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return cliArgs{}, fmt.Errorf("unknown flag: %s", arg)
			}
			if pathSet {
				return cliArgs{}, fmt.Errorf("multiple paths given")
			}
			a.path = arg
			pathSet = true
		}
	}
	return a, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(argv []string, stdout, stderr io.Writer) int {
	a, err := parseArgs(argv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	if a.showHelp {
		fmt.Fprint(stdout, helpText)
		return exitOK
	}
	if a.showVersion {
		fmt.Fprintln(stdout, version)
		return exitOK
	}

	repoRoot, err := git.RepoRoot(a.path)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}

	paths, err := git.ListGitignores(repoRoot)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}

	type fileWork struct {
		absPath string
		content string
		drop    map[int]bool
	}
	var fileResults []format.FileResult
	var works []fileWork
	pruneCount := 0

	for _, relPath := range paths {
		absPath := filepath.Join(repoRoot, relPath)
		contentBytes, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return exitUsage
		}
		content := string(contentBytes)
		gitignoreDir := filepath.Dir(absPath)

		var entries []format.Entry
		drop := map[int]bool{}
		for _, ln := range parse.Parse(content) {
			if ln.Kind != parse.KindEntry {
				continue
			}
			shouldPrune, err := git.ShouldPrune(gitignoreDir, ln.Text)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return exitUsage
			}
			status := format.Keep
			if shouldPrune {
				status = format.Prune
				drop[ln.LineNo] = true
				pruneCount++
			}
			entries = append(entries, format.Entry{Status: status, Text: ln.Text})
		}

		fileResults = append(fileResults, format.FileResult{Path: relPath, Entries: entries})
		works = append(works, fileWork{absPath: absPath, content: content, drop: drop})
	}

	if a.fix {
		for _, w := range works {
			if len(w.drop) == 0 {
				continue
			}
			newContent := rewrite.Apply(w.content, w.drop)
			if err := os.WriteFile(w.absPath, []byte(newContent), 0o644); err != nil {
				fmt.Fprintln(stderr, err)
				return exitUsage
			}
		}
		fmt.Fprint(stdout, format.Fix(fileResults))
		return exitOK
	}

	fmt.Fprint(stdout, format.Check(fileResults))
	if pruneCount > 0 {
		return exitPrunesFound
	}
	return exitOK
}
