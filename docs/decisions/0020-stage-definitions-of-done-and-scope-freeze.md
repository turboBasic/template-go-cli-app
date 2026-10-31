---
id: "0020"
status: accepted
date: 2026-10-31
scope: project
implementation_stage: All stages
---

# ADR 0020 — Stage definitions of done and scope freeze

## Decision

Freeze Stage 1, Stage 2 and Stage 3 as specified. Changes to stage scope require explicit ADR amendments. Every committed capability is implemented and tested in the template; external integrations remain runtime-optional.

## Context

Option A was selected: all three stages are committed. Option B would have kept Stage 3 provisional and allowed it to be redesigned before implementation. A Stage 3 commitment means capabilities are available in the template, not enabled or required in every generated CLI.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue. Related: [ADR 0001](0001-project-strategy.md).
