---
id: "0006"
status: accepted
date: 2026-10-31
scope: configuration
---

# ADR 0006 — Configuration discovery and precedence

## Decision

Load values by precedence: explicitly changed flags, environment, explicit or discovered project config, user config, then defaults. Discover project config only in the current working directory. An explicit `--config` replaces the discovered project file while retaining user config. Ignore missing optional files; error on a missing explicit file.

## Context

The accepted configuration contract specifies precedence, current-directory discovery and explicit project-file behavior.

## Options

Not saved.

## Consequences

Not saved.

## Links

No issue.
