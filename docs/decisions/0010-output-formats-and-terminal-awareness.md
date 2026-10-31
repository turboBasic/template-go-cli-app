---
id: "0010"
status: accepted
date: 2026-10-31
scope: cli
implementation_stage: Stages 1–2
---

# ADR 0010 — Output formats and terminal awareness

## Decision

Support text and JSON results. Keep results on stdout and diagnostics/progress on stderr. Detect terminal capabilities, provide color controls and degrade safely under redirection. Add richer formatting in Stage 2.

## Context

The accepted output contract requires machine-readable output and stream separation. YAML and CSV are explicitly excluded.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
