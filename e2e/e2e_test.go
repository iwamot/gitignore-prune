// Package e2e exercises the compiled gitignore-prune binary as a subprocess.
// Unit and integration tests in other packages call run() directly; these
// tests build the actual artifact and verify exit codes, stdout/stderr
// routing, and on-disk effects through a real os.Exec boundary.
package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iwamot/gitignore-prune/internal/testutil"
)

var binPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "gitignore-prune-e2e-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "TestMain: mkdir:", err)
		os.Exit(2)
	}
	binPath = filepath.Join(tmp, "gitignore-prune")
	out, buildErr := exec.Command("go", "build", "-o", binPath, "..").CombinedOutput()
	if buildErr != nil {
		fmt.Fprintf(os.Stderr, "TestMain: go build failed: %v\n%s", buildErr, out)
		os.RemoveAll(tmp)
		os.Exit(2)
	}
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

type result struct {
	stdout   string
	stderr   string
	exitCode int
}

func runBin(t *testing.T, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	if err == nil {
		return result{stdout: so.String(), stderr: se.String(), exitCode: 0}
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return result{stdout: so.String(), stderr: se.String(), exitCode: ee.ExitCode()}
	}
	t.Fatalf("run %v: %v", args, err)
	return result{}
}

func TestE2E_help(t *testing.T) {
	r := runBin(t, "--help")
	if r.exitCode != 0 {
		t.Errorf("exit = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, "Usage:") {
		t.Errorf("stdout missing Usage:\n%s", r.stdout)
	}
	if r.stderr != "" {
		t.Errorf("stderr should be empty, got: %q", r.stderr)
	}
}

func TestE2E_helpShortFlag(t *testing.T) {
	r := runBin(t, "-h")
	if r.exitCode != 0 {
		t.Errorf("exit = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, "Usage:") {
		t.Errorf("stdout missing Usage:\n%s", r.stdout)
	}
}

func TestE2E_version(t *testing.T) {
	r := runBin(t, "--version")
	if r.exitCode != 0 {
		t.Errorf("exit = %d, want 0", r.exitCode)
	}
	if strings.TrimSpace(r.stdout) == "" {
		t.Error("version output empty")
	}
	if r.stderr != "" {
		t.Errorf("stderr should be empty, got: %q", r.stderr)
	}
}

func TestE2E_unknownFlag(t *testing.T) {
	r := runBin(t, "--bogus")
	if r.exitCode != 2 {
		t.Errorf("exit = %d, want 2", r.exitCode)
	}
	if r.stderr == "" {
		t.Error("expected error message on stderr")
	}
	if r.stdout != "" {
		t.Errorf("stdout should be empty on usage error, got: %q", r.stdout)
	}
}

func TestE2E_notInRepo(t *testing.T) {
	dir := t.TempDir()
	r := runBin(t, dir)
	if r.exitCode != 2 {
		t.Errorf("exit = %d, want 2", r.exitCode)
	}
	if r.stderr == "" {
		t.Error("expected error message on stderr for non-repo")
	}
}

func TestE2E_emptyRepo(t *testing.T) {
	repo := testutil.SetupRepo(t)
	r := runBin(t, repo)
	if r.exitCode != 0 {
		t.Errorf("exit = %d, want 0 (stderr: %s)", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "No prunable entries found.") {
		t.Errorf("stdout missing summary:\n%s", r.stdout)
	}
}

func TestE2E_defaultModeFindsPrunes(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "*.log\nnonexistent/\n")
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	r := runBin(t, repo)
	if r.exitCode != 1 {
		t.Errorf("exit = %d, want 1 (stderr: %s)", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "[PRUNE] nonexistent/") {
		t.Errorf("stdout missing PRUNE line:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "[KEEP]  *.log") {
		t.Errorf("stdout missing KEEP line:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "Run with --fix") {
		t.Errorf("stdout missing summary:\n%s", r.stdout)
	}
}

func TestE2E_defaultModeAllKeep(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "*.log\n")
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	r := runBin(t, repo)
	if r.exitCode != 0 {
		t.Errorf("exit = %d, want 0 (stderr: %s)", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "No prunable entries found.") {
		t.Errorf("stdout missing summary:\n%s", r.stdout)
	}
}

func TestE2E_fixModeRewritesFile(t *testing.T) {
	repo := testutil.SetupRepo(t)
	gitignorePath := filepath.Join(repo, ".gitignore")
	original := "# comment\n*.log\n\nbogus/\nkeep-me/\n"
	testutil.WriteFile(t, gitignorePath, original)
	testutil.WriteFile(t, filepath.Join(repo, "app.log"), "")
	testutil.WriteFile(t, filepath.Join(repo, "keep-me", "x"), "")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	r := runBin(t, "--fix", repo)
	if r.exitCode != 0 {
		t.Errorf("exit = %d, want 0 (stderr: %s)", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "Pruned 1 entry.") {
		t.Errorf("stdout missing summary:\n%s", r.stdout)
	}

	got, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}
	want := "# comment\n*.log\n\nkeep-me/\n"
	if string(got) != want {
		t.Errorf("rewritten file mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestE2E_pathArgFromCwd(t *testing.T) {
	repo := testutil.SetupRepo(t)
	testutil.WriteFile(t, filepath.Join(repo, ".gitignore"), "nonexistent/\n")
	testutil.RunGit(t, repo, "add", ".gitignore")
	testutil.RunGit(t, repo, "commit", "-q", "-m", "init")

	cmd := exec.Command(binPath)
	cmd.Dir = repo
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != 1 {
		t.Errorf("expected exit 1 from cwd-based invocation, got err=%v stdout=%q stderr=%q", err, so.String(), se.String())
	}
	if !strings.Contains(so.String(), "[PRUNE] nonexistent/") {
		t.Errorf("stdout missing PRUNE line:\n%s", so.String())
	}
}
