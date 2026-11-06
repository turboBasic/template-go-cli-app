---
id: "0007"
status: accepted
date: 2026-10-31
scope: configuration
---

# ADR 0007 — Strict configuration and path resolution

## Decision

Reject unknown TOML keys and invalid types. Merge maps recursively and replace arrays. Resolve TOML path values against the declaring file; resolve flag and environment path values against the working directory. Retain source metadata required for path resolution.

## Context

The accepted contract requires strict validation and source-relative paths. Exact loader implementation remains an implementation detail.

## Options

Not saved.

## Consequences

Not saved.

## Links

No issue.
