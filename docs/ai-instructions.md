# Go CLI App Template project instructions

Go CLI App Template is GitHub template repository.

## Constitution

- Follow [Architecture Specification](architecture-specification.md) and the accepted
  [ADRs](decisions/README.md) for CLI contracts. Delivery scope lives in GitHub milestones and issues.
- This is a template: carry the minimum code and tests that demonstrate each contract
  ([ADR-0021](decisions/0021-the-template-carries-the-minimum-code-that-demonstrates-each-contract.md)).
- Known departures from them are tracked in [Technical debt](technical-debt.md).

## Stack and structure

- Go CLI; one Go module for the deployable binary.
- `mise.toml` pins local tools; `justfile` is the task interface; `prek` owns lint and formatting hooks.

## Go rules

- Return actionable errors and wrap causes with `%w`; do not panic for operational failures.
- Pass dependencies explicitly; do not add mutable package-level state.
- Format with `gofmt` and `goimports`. Exported identifiers need useful doc comments.
- For readability, manually wrap long function calls and declarations across lines; `gofmt` preserves these breaks.
- Put tests beside the code and use table-driven cases where behaviour has multiple cases.
- Keep one `go.mod` for this binary. Do not add dependencies for small standard-library tasks.

## Security and safety

- Use explicitly configured secret sources, including approved optional providers. Resolve secrets only
  when needed.
- Never put secret values in command-line arguments, logs, generated output, or configuration inspection.
- Keep provider credentials out of repository files; use environment variables or workload identity in CI.

## Dependency updates

- Renovate owns version updates. Dependabot owns security updates and alerts only.

## Repository workflow

- Run `just lint` after code or configuration changes, `just test` for behavior changes, and `just build`
  for build or toolchain changes.
- Application command groups with children are help-only; register operations on leaves and create a fresh registration and root per standalone execution. See [ADR-0024](decisions/0024-application-composition-owns-selected-command-lifecycle.md).
