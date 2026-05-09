# gitignore-prune

[![Validate](https://github.com/iwamot/gitignore-prune/actions/workflows/validate.yml/badge.svg)](https://github.com/iwamot/gitignore-prune/actions/workflows/validate.yml)
[![codecov](https://codecov.io/gh/iwamot/gitignore-prune/graph/badge.svg)](https://codecov.io/gh/iwamot/gitignore-prune)
[![Go](https://img.shields.io/github/go-mod/go-version/iwamot/gitignore-prune)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Detect `.gitignore` entries that no longer match anything in the working tree.

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
[PRUNE] *.log
[KEEP]  node_modules/
[PRUNE] coverage/
[KEEP]  dist/

Run with --fix to prune entries marked [PRUNE].
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | No prunable entries (or `--fix` succeeded) |
| `1`  | Prunable entries found (without `--fix`) |
| `2`  | Path is not inside a git repository, usage error, or `git` not on PATH |

## Why

A `.gitignore` usually starts overbroad — copied from a generator or
framework default, or assembled defensively to cover IDE files and build
outputs you might one day produce. Running the project for a while reveals
which patterns actually match files you generate; the rest is noise.

`gitignore-prune` checks each entry against the current working tree so you
can drop what isn't doing anything.

## How it works

For each tracked `.gitignore` in the repository, gitignore-prune writes each
entry into a temp file and asks git via `git ls-files --exclude-from=<tmp>`
whether the pattern matches any path in the working tree. If at least one
path (tracked or untracked) matches, the entry is `[KEEP]`. If nothing
matches, it's `[PRUNE]`.

Pattern matching is git's own, so the verdict is byte-identical to git's
`.gitignore` interpretation. Negation patterns (`!foo`) have the leading `!`
stripped before the check — `!foo` and `foo` name the same set of paths.

## Scope

- Scans every `.gitignore` tracked by git in the repository, including those
  in subdirectories.
- Each `.gitignore` is checked in its own scope: patterns in
  `subdir/.gitignore` are matched only against files under `subdir/`.
- Submodules are skipped — their `.gitignore` lives in the submodule's own
  index. Run gitignore-prune inside the submodule to prune those entries.
- `--fix` removes only `[PRUNE]` lines. Comments, blank lines, entry order,
  and line endings (LF / CRLF) are kept verbatim.

## Known limitations

- `.git/info/exclude` is not scanned. It's local-only and never committed,
  so it doesn't accumulate cruft the way tracked `.gitignore` files do.
- Entries are judged by what's in the working tree right now. Patterns for
  files generated only at certain times — e.g. `dist/` during release
  builds — get flagged as `[PRUNE]` when the files aren't present. Run
  `--fix` while the generated files exist, or skim the report and apply by
  hand.
- Negation pairs aren't modeled. `!foo` is evaluated on its own, not as
  half of a `foo` / `!foo` pair.
- An empty `.gitignore` isn't deleted by `--fix`. Remove it yourself if no
  entries remain.

## License

MIT
