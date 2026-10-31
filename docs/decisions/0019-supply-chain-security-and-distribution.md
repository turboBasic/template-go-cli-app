---
id: "0019"
status: accepted
date: 2026-10-31
scope: security
implementation_stage: Stage 3
---

# ADR 0019 — Supply-chain security and distribution

## Decision

Add SPDX JSON SBOMs, keyless Sigstore/Cosign signing of release checksums, GitHub provenance attestations, automated Homebrew tap publishing and artifact-verification tests. Keep Stage 2 GoReleaser/GitHub Releases unchanged.

## Context

The selected scope is the stronger keyless-signing option and Homebrew tap. Sigstore does not imply Apple Developer ID signing or notarization; reproducible builds are not committed requirements.

## Options

Not saved.

## Consequences

- This decision is part of the frozen Architecture Specification v1.0 baseline.
- Implementations must preserve the stated contract and the stage boundary.
- Any scope change requires an explicit ADR amendment under ADR-0020.

## Links

No issue.
