# Go CLI App Template — Roadmap

**Baseline:** Architecture Specification v1.0; ADR-001 through ADR-020 accepted
**Status:** Stage scope frozen. Amendments require an explicit ADR amendment.
**Platforms:** Linux and macOS, AMD64 and ARM64

The stages are sequential and independently verifiable. Stage 1 establishes the foundation; Stage 2 adds terminal UX and releases without changing application semantics; Stage 3 adds advanced execution, optional integrations and verifiable distribution. Stage 3 capabilities are mandatory in the template, while external integrations remain runtime-optional in generated applications.

## Stage 1 — Foundation

**Goal:** A generated repository contains a functional, testable, non-interactive-safe Go CLI.

**Committed scope:** GitHub template, mise initialization tasks and Prek hooks; pragmatic hexagonal architecture; simulated infrastructure CLI; Cobra hierarchy/help/completions/flags/version; Koanf v2 with TOML, strict layered configuration and source-relative paths; slog with stderr and optional JSONL file destinations and structured redaction; text/JSON output, stream separation, terminal detection and color controls; semantic progress interface and plain stderr reporter; classified errors, cancellation, cleanup and bounded infrastructure retries; four target builds and CI quality gates.

**Definition of done:** A newly instantiated project builds on Linux/macOS AMD64/ARM64, passes agreed tests and gates, and runs simulated resource commands end to end. JSON remains parseable with progress and logging enabled. Configuration precedence, secret masking, cancellation and exit behavior have automated coverage.

## Stage 2 — User experience and releases

**Goal:** Add rich terminal presentation and automated binary releases without changing application semantics.

**Committed scope:** Fang help/error presentation; Lip Gloss styling and formatted output; Bubbles/Bubble Tea animated progress; terminal-aware degradation and enhanced console logging; per-leaf configuration provenance; `config show` and `config explain`, key filtering and masked text/JSON output; GoReleaser and semantic Git-tagged GitHub Releases with archives, checksums and changelogs; golden and terminal behavior tests.

**Definition of done:** Interactive progress works; redirected output is safe and deterministic; inspection reports effective sources without revealing secrets; a tagged release publishes all four target binaries.

## Stage 3 — Advanced execution and security

**Goal:** Support advanced concurrent workflows, optional integrations and verifiable distribution.

**Committed scope:** Concurrent/nested progress with coordinated rendering; optional lazy 1Password CLI adapter behind a provider-independent interface; extensible remote logging sinks and optional built-in rotation; SPDX SBOMs; keyless Cosign checksum signing; GitHub provenance attestations; automated Homebrew tap publication; stress, integration and verification tests.

**Definition of done:** Concurrent simulated operations report progress without corrupting output or indefinitely blocking execution; secrets resolve on demand and remain out of inspection output; artifacts have verifiable signatures, provenance and SBOMs and can be installed using Homebrew.

## Scope control

All three stages are committed. Option A in ADR-020 was selected. Option B would have committed Stages 1–2 while leaving Stage 3 provisional. Under the selected baseline, changes to any stage scope require an explicit ADR amendment. Optional means runtime-optional for the generated application; it does not remove the capability from the template deliverable.

## Explicit exclusions

Windows; YAML/CSV output; runtime plugin framework; mandatory DI framework; configuration hot reload; automatic whole-workflow retries; multiple concrete secret providers; Debian/RPM repositories; byte-for-byte reproducible builds; macOS notarization; and mandatory numerical coverage threshold.
