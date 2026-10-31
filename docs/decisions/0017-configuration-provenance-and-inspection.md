---
id: "0017"
status: accepted
date: 2026-10-31
scope: configuration
implementation_stage: Stage 2
---

# ADR 0017 — Configuration provenance and inspection

## Decision

Track configuration source per leaf and provide `config show` and `config explain`, including key filtering and text/JSON forms. Mask sensitive values and never resolve secret references during inspection.

## Context

The accepted register ties provenance and inspection to Stage 2. A metadata-aware loading layer may be needed around Koanf; this is an implementation caution, not a new framework decision.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
