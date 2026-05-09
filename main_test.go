package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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
		{"version short", []string{"-V"}, cliArgs{showVersion: true, path: "."}, false},
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

func TestRun_help(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-h"}, &out, &errBuf)
	if code != exitOK {
		t.Errorf("exit = %d, want %d", code, exitOK)
	}
	if !bytes.Contains(out.Bytes(), []byte("Usage:")) {
		t.Errorf("help output missing Usage: section: %s", out.String())
	}
}

func TestRun_version(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"--version"}, &out, &errBuf)
	if code != exitOK {
		t.Errorf("exit = %d, want %d", code, exitOK)
	}
	if out.Len() == 0 {
		t.Error("version output empty")
	}
}

func TestRun_usageError(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"--bogus"}, &out, &errBuf)
	if code != exitUsage {
		t.Errorf("exit = %d, want %d", code, exitUsage)
	}
}

func TestRun_notInRepo(t *testing.T) {
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	code := run([]string{dir}, &out, &errBuf)
	if code != exitUsage {
		t.Errorf("exit = %d, want %d", code, exitUsage)
	}
}

func TestRun_emptyRepo(t *testing.T) {
	repo := testutil.SetupRepo(t)
	var out, errBuf bytes.Buffer
	code := run([]string{repo}, &out, &errBuf)
	if code != exitOK {
		t.Errorf("exit = %d, want %d (stderr: %s)", code, exitOK, errBuf.String())
	}
}

func TestRun_checkModeFindsPrunes(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "*.log\nnonexistent/\n")
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	var out, errBuf bytes.Buffer
	code := run([]string{repo}, &out, &errBuf)
	if code != exitPrunesFound {
		t.Errorf("exit = %d, want %d (stderr: %s)", code, exitPrunesFound, errBuf.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("[PRUNE] nonexistent/")) {
		t.Errorf("output missing PRUNE line: %s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("[KEEP]  *.log")) {
		t.Errorf("output missing KEEP line: %s", out.String())
	}
}

func TestRun_checkModeAllKeep(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "*.log\n")
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	var out, errBuf bytes.Buffer
	code := run([]string{repo}, &out, &errBuf)
	if code != exitOK {
		t.Errorf("exit = %d, want %d (stderr: %s)", code, exitOK, errBuf.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("No prunable entries found.")) {
		t.Errorf("output missing 'No prunable entries found.': %s", out.String())
	}
}

func TestRun_fixMode(t *testing.T) {
	repo := testutil.SetupRepo(t)
	original := "# top comment\n*.log\n\nnonexistent/\nkeep-me/\n"
	gitignorePath := filepath.Join(repo, ".gitignore")
	testutil.WriteFile(t, gitignorePath, original)
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.WriteFile(t, filepath.Join(repo, "keep-me", "x"), "")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	var out, errBuf bytes.Buffer
	code := run([]string{"--fix", repo}, &out, &errBuf)
	if code != exitOK {
		t.Errorf("exit = %d, want %d (stderr: %s)", code, exitOK, errBuf.String())
	}

	got, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}
	want := "# top comment\n*.log\n\nkeep-me/\n"
	if string(got) != want {
		t.Errorf("rewritten file mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if !bytes.Contains(out.Bytes(), []byte("Pruned 1 entry.")) {
		t.Errorf("summary missing: %s", out.String())
	}
}

func TestRun_fixModePreservesCRLF(t *testing.T) {
	repo := testutil.SetupRepo(t)
	original := "*.log\r\nnonexistent/\r\n"
	gitignorePath := filepath.Join(repo, ".gitignore")
	testutil.WriteFile(t, gitignorePath, original)
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	var out, errBuf bytes.Buffer
	code := run([]string{"--fix", repo}, &out, &errBuf)
	if code != exitOK {
		t.Errorf("exit = %d, want %d (stderr: %s)", code, exitOK, errBuf.String())
	}
	got, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}
	want := "*.log\r\n"
	if string(got) != want {
		t.Errorf("CRLF preservation failed\n--- got ---\n%q\n--- want ---\n%q", got, want)
	}
}

func TestRun_subdirGitignore(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "*.log\n")
	testutil.WriteFile(t, filepath.Join(repo, "subdir", ".gitignore"), "tmp/\nbogus/\n")
	testutil.WriteFile(t, filepath.Join(repo, "subdir", "tmp", "x"), "")
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.RunGit(t, repo, "add", ".gitignore", "subdir/.gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	var out, errBuf bytes.Buffer
	code := run([]string{repo}, &out, &errBuf)
	if code != exitPrunesFound {
		t.Errorf("exit = %d, want %d (stderr: %s)", code, exitPrunesFound, errBuf.String())
	}
	output := out.String()
	if !bytes.Contains(out.Bytes(), []byte("[PRUNE] bogus/")) {
		t.Errorf("expected subdir PRUNE entry: %s", output)
	}
	if !bytes.Contains(out.Bytes(), []byte("[KEEP]  tmp/")) {
		t.Errorf("expected subdir KEEP entry: %s", output)
	}
	if !bytes.Contains(out.Bytes(), []byte("[KEEP]  *.log")) {
		t.Errorf("expected root KEEP entry: %s", output)
	}
}
