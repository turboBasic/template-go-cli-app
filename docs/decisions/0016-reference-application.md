---
id: "0016"
status: accepted
date: 2026-10-31
scope: architecture
implementation_stage: Stages 1–3
---

# ADR 0016 — Reference application

## Decision

Include a simulated infrastructure CLI using local fixtures and no external credentials. Demonstrate representative `resource list`, `resource status` and `resource apply` flows through the same application use cases across stages.

## Context

The accepted register selects a simulated infrastructure reference application; later stages reuse the same use cases to demonstrate presentation and concurrent progress.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
