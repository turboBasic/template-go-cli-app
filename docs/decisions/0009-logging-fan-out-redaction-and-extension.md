---
id: "0009"
status: accepted
date: 2026-10-31
scope: observability
implementation_stage: Stages 1 and 3
---

# ADR 0009 — Logging fan-out, redaction and extension

## Decision

Apply structured secret redaction before fan-out, including known sensitive keys, nested attributes and secret-aware types. Stage 3 adds extensible remote sinks and optional built-in rotation. Do not intentionally log resolved secrets.

## Context

The accepted register associates JSONL, external rotation and structured redaction with the logging decisions. Free-form messages cannot be guaranteed safe by structured redaction alone.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
