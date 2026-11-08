---
id: "0022"
status: accepted
date: 2026-11-08
scope: cli
---

# ADR 0022 — Reference commands are demos named for the contract they demonstrate

## Decision

Every reference command lives under `cli-app demo` and is named for the one contract it demonstrates:
`demo structured-output`, `demo not-found <name>`, `demo secret-credential <name>` and `demo long-running`.
A demo's configuration keys and `CLI_APP_*` variables carry the command's name, with `-` written as `_`, under
a `demo` group (`timeouts.demo.<name>`, `retry.demo.<name>`, `demo.<name>.<setting>`). No alias keeps an old
name alive. `config show`, `completion` and `help` are the application's own contract and stay outside the group.

## Context

ADR-0016 named the reference flows after the simulated domain, as the `resource` group's `list`, `status`
and `apply`. Those names say what the fixture holds and nothing about which contract each command shows,
so a reader copying one could not tell which part was the lesson. The `demo` group from #83 already holds
`long-running`, named for its contract. The timeout key was derived from the command name in two places
(`internal/cli/root.go`), so a command rename alone would have silently dropped `--timeout` and the
configured timeouts. Evidence is in #84.

## Options

### Rename commands and keys together under `demo` (SELECTED)

- Adopted because: one name per setting, and the key reads as the command it configures.
- Adopted despite: a breaking change for scripts, configuration files and environment variables.

### Keep the `resource` group (ADR-0016)

- Rejected because: the names stay silent about the contract each command demonstrates.
- Rejected despite: nothing breaks and the domain vocabulary stays consistent.

### Rename the commands and leave the keys (the issue's default)

- Rejected because: the keys would name commands that no longer exist, or the derivation would need a table.
- Rejected despite: configuration files and variables keep working.

### Rename with aliases for the old commands and keys

- Rejected because: a second spelling keeps two names of one contract alive in a template that is copied.
- Rejected despite: scripts get a deprecation window.

## Consequences

- A command that only demonstrates a contract is added under `demo` with that contract's name, never under a
  domain noun. The `Short` of each demo begins `Demonstrate`, pinned by `TestDemoCommands`.
- An old key in a configuration file fails with an unknown-key error and exit 2. An old `CLI_APP_*` variable is
  ignored silently, because the loader skips a variable it does not know.
- The internal names (`ResourceService`, the `memory` fixture) stay: the simulated domain is what the demos run on.
- Reopen if an application built from the template needs a command outside `demo` to demonstrate a contract.

## Links

Issue #84. Supersedes [ADR 0016](0016-reference-application.md); sits beside
[ADR 0021](0021-the-template-carries-the-minimum-code-that-demonstrates-each-contract.md).
