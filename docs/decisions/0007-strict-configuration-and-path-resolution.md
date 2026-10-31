---
id: "0007"
status: accepted
date: 2026-10-31
scope: configuration
implementation_stage: Stage 1
---

# ADR 0007 — Strict configuration and path resolution

## Decision

Reject unknown TOML keys and invalid types. Merge maps recursively and replace arrays. Resolve TOML path values against the declaring file; resolve flag and environment path values against the working directory. Retain source metadata required for path resolution.

## Context

The accepted contract requires strict validation and source-relative paths. Exact loader implementation remains an implementation detail.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
