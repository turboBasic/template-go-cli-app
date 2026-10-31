---
id: "0018"
status: accepted
date: 2026-10-31
scope: security
implementation_stage: Stage 3
---

# ADR 0018 — Secret provider and resolution

## Decision

Define a provider-independent secret interface and implement an optional 1Password CLI (`op`) adapter supporting `op://` references. Resolve lazily on demand, do not persist resolved values, do not intentionally log them, and do not resolve them in config inspection. Support cancellation, distinguish authentication failures from missing secrets and avoid passing secret values in command-line arguments.

## Context

The accepted decision makes 1Password the initial optional provider and excludes additional provider implementations. Authentication is delegated to the configured `op` environment.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
