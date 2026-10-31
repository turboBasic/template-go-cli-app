---
id: "0014"
status: accepted
date: 2026-10-31
scope: delivery
implementation_stage: Stages 1–3
---

# ADR 0014 — Versioning and release automation

## Decision

Embed build/version metadata in Stage 1. Use semantic Git tags and GoReleaser with GitHub Actions in Stage 2 to publish release archives, checksums and changelogs for Linux/macOS AMD64/ARM64.

## Context

The accepted register selects semantic versions and GoReleaser in Stage 2. Stage 2 release behavior is unchanged by ADR-019.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
