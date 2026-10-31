---
id: "0013"
status: accepted
date: 2026-10-31
scope: quality
implementation_stage: Stages 1–3
---

# ADR 0013 — CI and quality gates

## Decision

Use production-oriented CI and quality checks across supported platforms. Run `go vet`, `golangci-lint` and race tests as specified for the stages. Measure coverage without imposing a numerical minimum.

## Context

The accepted register states production-oriented CI and coverage without a threshold.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
