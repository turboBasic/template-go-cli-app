---
id: "0023"
status: accepted
date: 2026-11-09
scope: cli
---

# ADR 0023 — Application commands register behind the shared CLI runtime

## Decision

The executable composes the shared root with one explicit `CommandRegistration`. The root owns shared runtime setup and teardown; the registered application owns its static command tree, typed settings, flag mappings, timeout policy, result text formatting, and lazy dependency construction. A selected command registers dependency cleanup with the root.

The shared renderer accepts an application-owned result and a text callback that receives the shared writer and text options. JSON encodes the result normally. Application settings are registered with typed defaults and validation; the loader validates the union of shared and registered schemas, preserving provenance and rejecting unknown TOML keys.

Retry remains an explicit application composition choice at a safe read dependency. The status-reader adapter is not constructed by the root and must not wrap workflows or mutations.

## Context

The previous root constructor required `*app.ResourceService`, built the demo commands, and selected demo-specific flags and operation timeouts. Its renderer understood only demo-domain resources. The #107 spike preferred explicit registration over a composition facade; #108 selected the corresponding result and retry boundaries before implementation.

## Consequences

- An application can replace the demo group without importing its service or domain types while retaining shared root behavior.
- Registration builds a static Cobra tree; selected commands initialize dependencies lazily and register cleanup, including cleanup after partial setup.
- Help, completion, and `config show` do not construct application dependencies or resolve secrets.
- Removing an application group also removes its schema and example settings. Unknown TOML keys remain errors; unknown environment variables remain ignored.
- ADR-0022 continues to name the reference demos and their settings. It does not make the demo group a shared-runtime requirement.

## Links

[Issue #108](https://github.com/turboBasic/template-go-cli-app/issues/108), following the [#107 composition spike](../research/command-composition-spike.md). ADR-0021 remains unchanged; ADR-0022 remains in force.
