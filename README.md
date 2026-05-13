# gitignore-prune

[![release](https://img.shields.io/github/v/release/iwamot/gitignore-prune)](https://github.com/iwamot/gitignore-prune/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/iwamot/gitignore-prune)](https://pkg.go.dev/github.com/iwamot/gitignore-prune)

Detect `.gitignore` entries that no longer match anything in the working tree.

A `.gitignore` usually starts overbroad — copied from a template, or
padded with patterns for files you might never actually produce.

## Usage

```bash
go install github.com/iwamot/gitignore-prune@latest
```

Or download a prebuilt binary from the [Releases page](https://github.com/iwamot/gitignore-prune/releases).

```bash
# Detect prunable entries
gitignore-prune                      # checks the repo containing the cwd
gitignore-prune path/to/repo         # or pass a path

# Remove prunable entries in place
gitignore-prune --fix
gitignore-prune --fix path/to/repo   # path and --fix can be combined
```

Example output:

```
=== .gitignore (4 entries) ===
[KEEP]  *.log
[KEEP]  !sample.log
[PRUNE] coverage/
[KEEP]  node_modules/

Run with --fix to prune entries marked [PRUNE].
```

## Caveat

Run `gitignore-prune` by hand, occasionally — not in pre-commit or CI.
Some entries are intentionally pre-emptive (`dist/` only built during
release, logs only one developer produces). `gitignore-prune` can't tell
those from real cruft, but a human reading the report can.

## Out of scope

- `.git/info/exclude` is not scanned. It's local-only and never committed,
  so it doesn't accumulate cruft.
- Submodules are skipped. Run `gitignore-prune` inside the submodule to
  prune its `.gitignore`.
- `--fix` doesn't touch comments, blank lines, entry order, or line
  endings — only `[PRUNE]` lines are removed.

## License

MIT
