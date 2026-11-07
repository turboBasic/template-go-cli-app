# Technical debt

Known departures from the architecture specification and the ADRs, each with a condition the repository can
evaluate. An entry is removed in the change that makes its condition true.

## Open entries

### TD-001 Help and errors follow Fang's color detection

`--color`, `CLI_APP_COLOR` and the config `color` key govern results only. Help and errors follow Fang's own
detection (color-capable TTY, `NO_COLOR`, `CLICOLOR_FORCE`, `TERM=dumb`), because forcing Fang to honor
`--color` needs a process-environment mutation and a second argument parser.

The two also differ on one input: with `CLICOLOR_FORCE=1` and `NO_COLOR=1` on a non-TTY, errors give
`NO_COLOR` precedence and print plain, while help ignores `NO_COLOR` and prints styled.

- Origin: S2-01 (#16); ADR-0010 requires color controls.
- Condition: both `env NO_COLOR=1 CLICOLOR_FORCE=1 TERM=xterm-256color dist/cli-app --help | rg -c '38;'` and
  `env CLICOLOR_FORCE=1 TERM=xterm-256color dist/cli-app --color never --help | rg -c '38;'` print nothing (`rg`
  prints no count when nothing matches; at HEAD both print 16).

### TD-003 `help <unknown>` succeeds silently

`cli-app help nope` prints the root help and exits 0 with nothing on stderr. Stage 1 printed
"Unknown help topic" to stderr. The change is deliberate: a root `Args` validator, which classifies unknown
commands as validation errors, stops Cobra's `Find` from rejecting unknown help topics (commit 586b1d0).

- Origin: S2-01 (#16); ADR-0010 sends diagnostics to stderr.
- Condition: `dist/cli-app help nope 2>&1 >/dev/null | rg -c .` prints a non-zero count.

### TD-004 The release workflow publishes a tag's commit unchecked

`.github/workflows/release.yml` runs `goreleaser release --clean` on any `vX.Y.Z` tag. It runs no tests and no
`scripts/check-release.sh` first, and `ci.yml` covers only `main` and pull requests, so a tag on a commit that
never went through CI publishes untested archives. The release snapshot job already catches a broken GoReleaser
configuration on every pull request; what remains is tagging an unchecked commit. Either run the tests and the
archive check inside `release.yml` before publishing, or restrict `v*` tags to commits on `main` with a ruleset.

- Origin: S2-05 (#20); ADR-0014 publishes releases from tags. No open issue owns it: S2-06 (#19) closed as not
  planned.
- Condition: `rg -c 'mise run test|check-release' .github/workflows/release.yml` prints a non-zero count (`rg`
  prints no count when nothing matches; at HEAD it prints nothing).
