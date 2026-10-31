---
id: "0008"
status: accepted
date: 2026-10-31
scope: observability
implementation_stage: Stage 1
---

# ADR 0008 — Local logging defaults and destinations

## Decision

Use Go `log/slog` with human-readable stderr logs defaulting to WARN and an optional JSON Lines file destination defaulting to DEBUG. Set destination levels independently. Do not create a log file by default; require an explicit path. Use restrictive permissions where supported, external rotation and fail initialization when a required destination cannot open.

## Context

These are the accepted production-oriented local logging requirements. No additional rationale is reconstructed.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
