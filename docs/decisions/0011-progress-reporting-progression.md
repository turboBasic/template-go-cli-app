---
id: "0011"
status: accepted
date: 2026-10-31
scope: cli
implementation_stage: Stages 1–3
---

# ADR 0011 — Progress reporting progression

## Decision

Define semantic progress independent of rendering and provide a plain stderr reporter in Stage 1; add animated progress in Stage 2; coordinate concurrent and nested progress in Stage 3.

## Context

This staged progression is accepted in the register and final scope. Rendering remains outside domain and application logic.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
