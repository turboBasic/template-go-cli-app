---
id: "0019"
status: accepted
date: 2026-10-31
scope: security
---

# ADR 0019 — Supply-chain security and distribution

## Decision

Sign release checksums with keyless Sigstore/Cosign. SPDX JSON SBOMs, GitHub provenance attestations and Homebrew tap publishing are optional release capabilities of the template.

## Context

The selected scope is the stronger keyless-signing option and Homebrew tap. Sigstore does not imply Apple Developer ID signing or notarization; reproducible builds are not committed requirements.

## Options

Not saved.

## Consequences

Not saved.

## Links

No issue.
