---
id: "0024"
status: accepted
date: 2026-11-10
scope: cli
---

# ADR 0024 — Application composition owns selected-command lifecycle policies

## Decision

Application composition provides one typed application configuration and binds group setup and timeout policies to the static Cobra command objects. The root resolves the nearest declared policy for the selected command, starts its operation deadline before setup, and runs registered cleanup once in reverse order under a fresh bounded context. Each standalone execution creates a fresh registration and root; application command groups with children are help-only, and operations run on leaf commands. Help and completion skip configuration loading, while configuration inspection loads and validates settings without setting up dependencies or resolving secrets.

## Context

The two-group fixture requested by [Issue #113](https://github.com/turboBasic/template-go-cli-app/issues/113) must prove that either selection leaves the other group uninitialized while one strict application schema validates both. The current demo builds dependencies through lazy handler calls and maps timeout settings through a command-name switch. Those paths do not express a group default, child override, and explicit no-setup policy at registration.

## Options

### Bind policies to Cobra command objects (SELECTED)

- Adopted because: application composition retains typed settings and declares setup and timeout behavior at registration.
- Adopted despite: group setup and child overrides must be attached while building the static tree.

### Construct dependencies lazily from each `RunE`

- Rejected because: shared setup, timeout, and cleanup behavior remains hidden inside handlers and group caches.
- Rejected despite: a small command can construct only what its handler immediately needs.

### Select application behavior by command name in the root

- Rejected because: the shared root would depend on application names and types instead of command objects.
- Rejected despite: one central callback is easy to add to a single-demo application.

## Consequences

- An absent setup declaration inherits from the nearest ancestor; a declared setup with no function explicitly skips setup. A nearer command policy overrides its group's default.
- `--timeout` overrides the configured selector. Setup and handler share its deadline; cleanup uses the configured cleanup bound and cleanup errors remain joined with operation errors.
- Build a new root tree for each execution. Register cleanup as soon as it is available, including when setup fails. Resolve secret values only for the operation that needs them.
- Build a fresh application registration and root for each standalone execution. Groups with children provide help only; the CLI installs their shared help and unknown-subcommand behavior, and application authors put operations on leaves.
- Help and completion do not load application configuration. `config show` loads and validates it but does not initialize services or resolve secrets. Existing keys, precedence, provenance, and ADR-0022 demo names remain in force.
- Reopen if a supported multi-execution or alternate composition contract needs to reuse a command tree.

## Links

[Issue #113](https://github.com/turboBasic/template-go-cli-app/issues/113). Sits beside [ADR 0018](0018-secret-provider-and-resolution.md) and [ADR 0022](0022-reference-commands-are-demos-named-for-their-contract.md).
