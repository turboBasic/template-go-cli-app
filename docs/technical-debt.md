# Technical debt

Known departures from the architecture specification and the ADRs, each with a condition the repository can
evaluate. An entry is removed in the change that makes its condition true.

## Open entries

### TD-003 `help <unknown>` succeeds silently

`cli-app help nope` prints the root help and exits 0 with nothing on stderr. Stage 1 printed
"Unknown help topic" to stderr. The change is deliberate: a root `Args` validator, which classifies unknown
commands as validation errors, stops Cobra's `Find` from rejecting unknown help topics (commit 586b1d0).

- Origin: S2-01 (#16); ADR-0010 sends diagnostics to stderr.
- Condition: `dist/cli-app help nope 2>&1 >/dev/null | rg -c .` prints a non-zero count.
