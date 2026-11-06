---
id: "0018"
status: accepted
date: 2026-10-31
scope: security
---

# ADR 0018 — Secret provider and resolution

## Decision

Define a provider-independent secret interface; provider adapters, the 1Password CLI (`op`) with `op://` references first, are optional. Resolve lazily on demand, do not persist resolved values, do not intentionally log them, and do not resolve them in config inspection. Support cancellation, distinguish authentication failures from missing secrets and avoid passing secret values in command-line arguments.

## Context

The accepted decision makes 1Password the initial optional provider and excludes additional provider implementations. Authentication is delegated to the configured `op` environment.

## Options

Not saved.

## Consequences

Not saved.

## Links

No issue.
