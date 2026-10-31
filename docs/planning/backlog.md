# Go CLI App Template — Staged Implementation Backlog

**Baseline:** Frozen Architecture Specification v1.0
**Ordering:** Dependencies first; work packages below are an implementation breakdown of accepted scope, not additional architecture decisions.
**Status:** Not started (initial export)

Task IDs and checklists are for execution tracking. Exact numeric exit-code mapping, default color flag spelling, environment-variable naming, external-rotation reopen policy, retry backoff values, exact Prek hook selection and dependency version pins remain implementation-level decisions to finalize without changing accepted contracts.

## Stage 1 — Foundation

### S1-01 — Template bootstrap and initialization

**Related ADRs:** 0001, 0015

- [ ] Create the GitHub template repository skeleton and Go module.
- [ ] Add mise-managed tool versions and initialization tasks.
- [ ] Add a setup flow that updates project name, module identity and configuration identifiers while leaving a runnable project.
- [ ] Add README setup and development instructions.

**Acceptance:** A fresh template instantiation can be initialized and built without manual source rewrites beyond documented setup.

### S1-02 — Architecture skeleton and composition

**Related ADRs:** 0002, 0016

- [ ] Establish only the needed CLI, application, domain and infrastructure boundaries.
- [ ] Wire dependencies explicitly during bootstrap; keep interfaces near consumers.
- [ ] Keep application/domain packages independent of Cobra, Koanf and terminal rendering.

**Acceptance:** A representative command reaches the simulated adapter through application logic without domain dependencies on CLI or external implementations.

### S1-03 — Cobra command and reference workflows

**Related ADRs:** 0003, 0004, 0016

- [ ] Add root command, help, version metadata, completions and standard global flags.
- [ ] Add `resource list`, `resource status` and `resource apply` using local simulated fixtures.
- [ ] Ensure automation-safe behavior; commands do not unexpectedly prompt.

**Acceptance:** The three reference commands run end to end without external credentials, and help/completions/version are available.

### S1-04 — Typed configuration loader and provenance foundation

**Related ADRs:** 0005, 0006, 0007, 0017

- [ ] Load TOML through Koanf v2 into typed configuration structs.
- [ ] Implement flag, environment, project, user and default precedence; only changed flags override.
- [ ] Discover project config only in the working directory; honor explicit `--config` while retaining user config.
- [ ] Ignore missing optional files; report a missing explicit config file.
- [ ] Reject unknown keys and invalid types; merge maps recursively and replace arrays.
- [ ] Resolve TOML paths against their declaring file and flag/environment paths against the working directory.
- [ ] Retain source metadata needed for path resolution and Stage 2 leaf provenance.

**Acceptance:** Automated tests cover precedence, discovery boundaries, explicit config behavior, strict decoding, map/array merging and path origins. Implementation avoids assuming Koanf itself provides all provenance metadata.

### S1-05 — Logging and structured redaction

**Related ADRs:** 0008, 0009, 0018

- [ ] Configure slog human-readable stderr output at WARN by default.
- [ ] Add optional explicitly configured JSONL file output at DEBUG with independent levels.
- [ ] Use restrictive file permissions where supported, external rotation and initialization errors for required destination failures.
- [ ] Redact known sensitive keys, nested attributes and secret-aware types before destination fan-out.

**Acceptance:** Tests prove destination-level behavior and structured redaction. No log file is created by default; failures opening a required destination fail startup.

### S1-06 — Output contract and terminal capabilities

**Related ADRs:** 0010

- [ ] Implement text and JSON result modes.
- [ ] Route results to stdout and diagnostics/logging/progress to stderr.
- [ ] Detect terminal capabilities and support color controls with safe redirected output.

**Acceptance:** Tests confirm JSON remains parseable and stdout contains no progress or diagnostic noise with logging/progress enabled.

### S1-07 — Progress, errors, cancellation and retry policy

**Related ADRs:** 0011, 0012

- [ ] Define semantic progress events and a plain stderr reporter.
- [ ] Add classified errors, context cancellation, cleanup and bounded retry behavior for eligible infrastructure operations only.
- [ ] Finalize numeric exit-code mapping and retry defaults as implementation details.

**Acceptance:** Tests cover cancellation, cleanup, failure classification, exit behavior and retries; whole workflows are not automatically retried.

### S1-08 — Hooks, CI, quality gates and target builds

**Related ADRs:** 0013, 0015

- [ ] Configure Prek fast hooks and align CI commands with local quality commands.
- [ ] Add `go vet`, `golangci-lint`, `go test -race` and relevant unit/integration checks.
- [ ] Build and smoke-test Linux/macOS AMD64/ARM64.
- [ ] Measure coverage without enforcing a numerical minimum.

**Acceptance:** A fresh generated project passes the agreed local and CI gates on all four targets and meets the Stage 1 definition of done.

## Stage 2 — User experience and releases

### S2-01 — Fang presentation integration

**Related ADRs:** 0003, 0004

- [ ] Integrate Fang for improved help and error presentation without changing command semantics.

**Acceptance:** Existing commands, flags, exit behavior and non-interactive use remain consistent; golden/help behavior is covered.

### S2-02 — Rich text and formatted output

**Related ADRs:** 0010

- [ ] Add Lip Gloss styling and rich text formatting.
- [ ] Preserve plain/deterministic behavior when output is redirected or color is disabled.

**Acceptance:** Golden tests cover representative formatted output and redirected output contains no unsafe terminal styling.

### S2-03 — Animated progress with safe degradation

**Related ADRs:** 0011

- [ ] Add Bubbles/Bubble Tea progress bars and spinners for interactive terminals.
- [ ] Degrade to safe output when non-interactive or redirected.
- [ ] Enhance console logging while preserving stdout/stderr ownership.

**Acceptance:** Terminal behavior tests cover interactive and redirected operation; JSON remains parseable.

### S2-04 — Per-leaf configuration provenance

**Related ADRs:** 0005, 0007, 0017

- [ ] Expose effective source metadata for each configuration leaf using the retained Stage 1 metadata.
- [ ] Implement `config show` and `config explain`, including key filtering.
- [ ] Support text and JSON inspection with sensitive values masked.
- [ ] Ensure inspection never resolves secret references.

**Acceptance:** Tests identify the correct effective source for layered values and demonstrate masking and non-resolution in both output modes.

### S2-05 — GoReleaser and tagged GitHub Releases

**Related ADRs:** 0014

- [ ] Configure GoReleaser and GitHub Actions for semantic Git-tagged releases.
- [ ] Publish archives, checksums and changelogs for Linux/macOS AMD64/ARM64.

**Acceptance:** A release workflow from a test tag produces all four targets and expected release metadata.

### S2-06 — Stage 2 acceptance and release verification

**Related ADRs:** 0010, 0011, 0013, 0014, 0017

- [ ] Add golden, terminal capability and animated-progress tests.
- [ ] Verify Stage 1 output and command contracts remain unchanged.

**Acceptance:** The example CLI is safe and deterministic under redirection, configuration inspection masks values, and a tagged release publishes all four target binaries. Stage 2 definition of done is met.

## Stage 3 — Advanced execution and security

### S3-01 — Concurrent and nested progress coordination

**Related ADRs:** 0011, 0016

- [ ] Coordinate concurrent/nested progress events and terminal rendering.
- [ ] Extend the simulated reference application to exercise multiple operations.

**Acceptance:** Concurrent operations do not corrupt output or indefinitely block application execution; stress tests cover expected concurrency.

### S3-02 — Secret provider contract and lazy resolution

**Related ADRs:** 0018

- [ ] Add provider-independent secret resolver interface.
- [ ] Resolve references only when a use case requests them; do not persist resolved values.
- [ ] Keep config inspection independent of resolution; ensure resolved values are not intentionally logged.

**Acceptance:** Tests prove lazy behavior, non-persistence at the application layer, inspection non-resolution, cancellation and error distinction.

### S3-03 — Optional 1Password CLI adapter

**Related ADRs:** 0018

- [ ] Implement the optional adapter for `op://` references using the configured 1Password CLI environment.
- [ ] Avoid passing resolved secret values as command-line arguments.
- [ ] Keep the adapter unavailable unless explicitly configured/used so `--help` and ordinary operation do not require `op` or an account.

**Acceptance:** Integration tests cover successful resolution, missing secrets, authentication failure and cancellation. The CLI remains usable without 1Password installed when the integration is unused.

### S3-04 — Remote logging and optional rotation capability

**Related ADRs:** 0009

- [ ] Add extensible remote logging sink support.
- [ ] Add optional built-in rotation capability while preserving the Stage 1 no-default-file and explicit-destination behavior.

**Acceptance:** Sink errors and rotation behavior are tested without changing existing destination defaults; integrations are optional at runtime.

### S3-05 — SBOM, checksum signing and provenance

**Related ADRs:** 0014, 0019

- [ ] Generate SPDX JSON SBOMs for release artifacts.
- [ ] Apply keyless Sigstore/Cosign signatures to release checksums.
- [ ] Generate GitHub provenance attestations.
- [ ] Keep Stage 2 GoReleaser/GitHub Releases behavior intact.

**Acceptance:** Release verification checks confirm the SBOM format and verify signatures and provenance for the expected artifacts. Unsupported environments are handled without implying Apple notarization or reproducible builds.

### S3-06 — Automated Homebrew tap

**Related ADRs:** 0019

- [ ] Publish Homebrew formula updates automatically from the release workflow.
- [ ] Keep direct GitHub downloads available.

**Acceptance:** A release can be installed through the configured tap on supported Homebrew environments and release metadata points to the correct artifacts.

### S3-07 — Stage 3 integration and concurrency tests

**Related ADRs:** 0011, 0018, 0019

- [ ] Add concurrency stress tests, optional secret-provider integration tests and artifact verification checks.
- [ ] Exercise the full simulated workflow with progress and configured integrations.

**Acceptance:** Tests demonstrate no output corruption, no indefinite progress blocking, no secret resolution during inspection and verifiable release artifacts.

### S3-08 — Final template acceptance and documentation

**Related ADRs:** 0013, 0015, 0020

- [ ] Document opt-in runtime configuration for external integrations and release capabilities.
- [ ] Verify Stage 1 and Stage 2 contracts still pass alongside Stage 3 checks.
- [ ] Confirm the template can be used without mandatory external accounts, remote log services or a Homebrew tap.

**Acceptance:** All three frozen stage definitions of done are met; every committed capability is implemented and tested in the template while external integrations remain runtime-optional. Any requested scope change is recorded as an explicit ADR amendment.

## Cross-stage notes

- Build and test all four supported targets throughout the stages.
- Keep user-facing implementation details out of domain/application layers.
- Do not add excluded features without an explicit scope amendment.
- Sequence work by dependencies; the recommended starting sequence is S1-01 and S1-02, then the smallest end-to-end `resource list` path before expanding cross-cutting components.
