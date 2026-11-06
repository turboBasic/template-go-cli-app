---
id: "0009"
status: accepted
date: 2026-10-31
scope: observability
---

# ADR 0009 — Logging fan-out, redaction and extension

## Decision

Apply structured secret redaction before fan-out, including known sensitive keys, nested attributes and secret-aware types. Do not intentionally log resolved secrets.

## Context

The accepted register associates JSONL, external rotation and structured redaction with the logging decisions. Free-form messages cannot be guaranteed safe by structured redaction alone.

## Options

Not saved.

## Consequences

Not saved.

## Links

No issue.
