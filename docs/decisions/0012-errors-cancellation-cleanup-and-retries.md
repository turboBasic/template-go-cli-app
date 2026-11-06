---
id: "0012"
status: accepted
date: 2026-10-31
scope: reliability
---

# ADR 0012 — Errors, cancellation, cleanup and retries

## Decision

Use classified errors, propagate cancellation, perform cleanup and retry only eligible infrastructure operations with bounded behavior. Do not automatically retry whole workflows.

## Context

The accepted scope explicitly includes these behaviors and excludes whole-workflow retries. Exact exit codes and retry defaults remain open implementation details.

## Options

Not saved.

## Consequences

Not saved.

## Links

No issue.
