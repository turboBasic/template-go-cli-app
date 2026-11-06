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
  `env CLICOLOR_FORCE=1 TERM=xterm-256color dist/cli-app --color never --help | rg -c '38;'` print 0.

### TD-002 Redirected help is space-padded

Fang pads redirected help lines with trailing spaces, so golden files are excluded from the whitespace hooks
(`trailing-whitespace`, `end-of-file-fixer`), `.editorconfig` and cspell. The sentinel
`internal/cli/testdata/whitespace.golden` fails the hook run if that exclusion is removed.

- Origin: S2-01 (#16).
- Condition: `rg -n ' $' internal/cli/testdata/*help*.golden` returns nothing.

### TD-003 `help <unknown>` succeeds silently

`cli-app help nope` prints the root help and exits 0 with nothing on stderr. Stage 1 printed
"Unknown help topic" to stderr. The change is deliberate: a root `Args` validator, which classifies unknown
commands as validation errors, stops Cobra's `Find` from rejecting unknown help topics (commit 586b1d0).

- Origin: S2-01 (#16); ADR-0010 sends diagnostics to stderr.
- Condition: `dist/cli-app help nope 2>&1 >/dev/null | rg -c .` prints a non-zero count.
