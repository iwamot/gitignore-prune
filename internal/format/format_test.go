package format

import "testing"

func TestCheck_empty(t *testing.T) {
	got := Check(nil)
	want := "No prunable entries found.\n"
	if got != want {
		t.Errorf("Check(nil) = %q, want %q", got, want)
	}
}

func TestCheck_singleFileWithPrunes(t *testing.T) {
	files := []FileResult{
		{
			Path: ".gitignore",
			Entries: []Entry{
				{Status: Prune, Text: "*.log"},
				{Status: Keep, Text: "node_modules/"},
			},
		},
	}
	want := `=== .gitignore (2 entries) ===
[PRUNE] *.log
[KEEP]  node_modules/

Run with --fix to prune entries marked [PRUNE].
`
	got := Check(files)
	if got != want {
		t.Errorf("Check() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCheck_multipleFiles(t *testing.T) {
	files := []FileResult{
		{
			Path: ".gitignore",
			Entries: []Entry{
				{Status: Prune, Text: "*.log"},
			},
		},
		{
			Path: "subdir/.gitignore",
			Entries: []Entry{
				{Status: Keep, Text: ".cache/"},
			},
		},
	}
	want := `=== .gitignore (1 entry) ===
[PRUNE] *.log

=== subdir/.gitignore (1 entry) ===
[KEEP]  .cache/

Run with --fix to prune entries marked [PRUNE].
`
	got := Check(files)
	if got != want {
		t.Errorf("Check() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCheck_noPrunes(t *testing.T) {
	files := []FileResult{
		{
			Path: ".gitignore",
			Entries: []Entry{
				{Status: Keep, Text: "node_modules/"},
			},
		},
	}
	want := `=== .gitignore (1 entry) ===
[KEEP]  node_modules/

No prunable entries found.
`
	got := Check(files)
	if got != want {
		t.Errorf("Check() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCheck_emptyEntries(t *testing.T) {
	files := []FileResult{
		{Path: ".gitignore", Entries: nil},
	}
	want := `=== .gitignore (0 entries) ===

No prunable entries found.
`
	got := Check(files)
	if got != want {
		t.Errorf("Check() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFix_empty(t *testing.T) {
	got := Fix(nil)
	want := "No prunable entries found.\n"
	if got != want {
		t.Errorf("Fix(nil) = %q, want %q", got, want)
	}
}

func TestFix_noPrunes(t *testing.T) {
	files := []FileResult{
		{
			Path: ".gitignore",
			Entries: []Entry{
				{Status: Keep, Text: "node_modules/"},
			},
		},
	}
	want := `=== .gitignore (1 entry) ===
[KEEP]  node_modules/

No prunable entries found.
`
	got := Fix(files)
	if got != want {
		t.Errorf("Fix() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFix_singlePrune(t *testing.T) {
	files := []FileResult{
		{
			Path: ".gitignore",
			Entries: []Entry{
				{Status: Prune, Text: "*.log"},
			},
		},
	}
	want := `=== .gitignore (1 entry) ===
[PRUNE] *.log

Pruned 1 entry.
`
	got := Fix(files)
	if got != want {
		t.Errorf("Fix() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFix_multiplePrunes(t *testing.T) {
	files := []FileResult{
		{
			Path: ".gitignore",
			Entries: []Entry{
				{Status: Prune, Text: "*.log"},
				{Status: Prune, Text: "*.tmp"},
				{Status: Keep, Text: "node_modules/"},
			},
		},
	}
	want := `=== .gitignore (3 entries) ===
[PRUNE] *.log
[PRUNE] *.tmp
[KEEP]  node_modules/

Pruned 2 entries.
`
	got := Fix(files)
	if got != want {
		t.Errorf("Fix() mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
