---
id: "0002"
status: accepted
date: 2026-10-31
scope: architecture
implementation_stage: Stage 1
---

# ADR 0002 — Architecture style

## Decision

Use pragmatic hexagonal architecture with dependency direction toward domain/application logic, explicit composition and replaceable adapters where useful. Avoid mandatory abstractions without a real boundary.

## Context

The architecture needs clear separation among CLI, application, domain and infrastructure. No additional rejected alternatives or rationale are reconstructed.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
