# Go CLI App Template — Architecture Specification v1.0

**Status:** Approved; scope frozen
**Decision baseline:** ADR-001 through ADR-020
**Implementation model:** Three sequential stages
**Target platforms:** Linux and macOS, AMD64 and ARM64
**Primary language:** Go

## 1. Purpose and objectives

Build a reusable GitHub template repository for modern Go command-line applications. Each use of the template creates an independent application; the template is not a shared runtime framework.

The template provides hierarchical commands, help, flags, shell completion and version reporting; separation of CLI presentation, application logic, domain logic and infrastructure; layered, strongly typed configuration; structured logging and secret redaction; text and JSON output; terminal-aware progress; cancellation and classified errors; automated quality checks; cross-platform releases; and the Stage 3 integration and supply-chain capabilities defined below.

## 2. Architectural principles

1. Use pragmatic hexagonal architecture: create boundaries where dependencies need to be replaceable or tested, without adding indirection without a use case.
2. Keep dependency direction inward. Domain logic does not depend on the CLI, presentation libraries or external-service implementations.
3. Construct and inject dependencies during application bootstrap.
4. Use typed Go contracts and classified errors rather than unstructured maps or implicit global state.
5. Keep command semantics and output contracts stable as presentation capabilities grow.
6. Make automation safe: results go to stdout; diagnostics, logs and progress go to stderr.
7. Treat secrets as sensitive values throughout configuration, logging and output.
8. Deliver the core first, then richer terminal UX and releases, then advanced integrations and supply-chain security.
9. Implement committed integrations as template capabilities while keeping external integrations runtime-optional.

## 3. Technology decisions

| Concern                          | Selected technology                                            | Stage |
| -------------------------------- | -------------------------------------------------------------- | ----- |
| CLI command framework            | Cobra                                                          | 1     |
| Rich CLI presentation            | Fang                                                           | 2     |
| Configuration                    | Koanf v2 with TOML                                             | 1     |
| Structured logging               | Go `log/slog` and custom handlers                              | 1     |
| Text and JSON output             | Go standard library and presentation adapters                  | 1     |
| Terminal capability detection    | Go terminal libraries                                          | 1     |
| Terminal styling                 | Lip Gloss                                                      | 2     |
| Animated progress                | Bubbles / Bubble Tea                                           | 2     |
| Concurrent progress coordination | Application event coordination                                 | 3     |
| Secret resolution                | Provider-independent interface; optional 1Password CLI adapter | 3     |
| Tool versions and task runner    | mise                                                           | 1     |
| Git hooks                        | Prek                                                           | 1     |
| Static analysis                  | `go vet`, `golangci-lint`                                      | 1     |
| CI                               | GitHub Actions                                                 | 1     |
| Release automation               | GoReleaser                                                     | 2     |
| Supply-chain verification        | Cosign, SPDX SBOM, GitHub attestations                         | 3     |
| Package distribution             | Automated Homebrew tap                                         | 3     |

Exact dependency versions are pinned during implementation and maintained under a dependency update policy; the baseline does not prescribe arbitrary version numbers.

## 4. Package boundaries

A reasonable layout is:

```text
cmd/cli-app/main.go
internal/cli/{root,resource,config}.go                  # Cobra commands and command wiring
internal/app/{resource,progress}.go                     # use cases and orchestration
internal/domain/{resource}.go                           # models, invariants and business rules
internal/config/{config,load,provenance,validate}.go    # loading, source metadata and validation
internal/logging/{setup,redact,fanout}.go               # slog setup, redaction and destinations
internal/output/{renderer,terminal,progress}.go         # result rendering, terminal capabilities and progress
internal/infra/{fixture,retry}/                         # external-facing adapters and simulated fixture
internal/secrets/{resolver.go,onepassword/}             # Stage 3 secret abstraction and 1Password adapter
test/fixtures/
docs/architecture-specification.md
docs/decisions/
scripts/init.sh
mise.toml
prek.toml
README.md
```

This layout is a starting point, not a requirement to create every package before it is needed. `cli` calls application use cases and presentation adapters; `app` coordinates domain operations through narrow consumer-owned interfaces; `domain` contains rules and models; `infra` implements external-facing interfaces; `config`, `logging` and `output` own their respective cross-cutting behavior. Application and domain code do not access Koanf directly or emit terminal escape sequences. A separate repository abstraction is not mandatory.

## 5. Configuration contract

**Format:** TOML. Values decode into typed structs and are validated before use. Unknown keys and invalid types are errors.

**Precedence, highest first:**

1. Explicit command-line flags
2. Environment variables
3. Explicit project configuration (`--config`) or discovered project configuration
4. User configuration
5. Built-in defaults

Configuration locations and discovery:

- User file: `$XDG_CONFIG_HOME/cli-app/config.toml`, defaulting to `~/.config/cli-app/config.toml` on Unix.
- Discovered project file: `.cli-app.toml` in the working directory only.
- `--config /path/to/custom.toml` selects the explicit project file and retains user configuration.
- Missing optional files are ignored; a missing explicitly named file is an error.
- Only explicitly changed flags override lower layers.
- Maps merge recursively; arrays are replaced.
- File-path-typed values from TOML resolve relative to the file that declares them. Values from flags or environment resolve relative to the working directory.

Stage 1 retains source metadata needed for path resolution. Stage 2 exposes per-leaf provenance through `config show` and `config explain`. Inspection reports effective values and source identities, supports text and JSON output and key filtering, masks sensitive values, and never resolves secret references. Per-key origin tracking and source-relative path resolution may require a metadata-aware loading layer around Koanf.

## 6. CLI and output contract

Use hierarchical lowercase kebab-case commands, consistent global flags, help, shell completion and version information. Non-interactive use must not unexpectedly prompt. Results go to stdout; diagnostics and progress go to stderr. JSON output remains parseable under normal operation and failures. Stage 1 supplies `text` and `json` modes, terminal detection, color controls and safe redirected output. Stage 2 adds Fang and Lip Gloss; Stage 3 adds coordinated concurrent progress. Domain and application code remain independent of rendering libraries.

The reference application is a simulated infrastructure CLI and uses local fixtures without external credentials. Representative commands are `cli-app resource list`, `cli-app resource status` and `cli-app resource apply`.

## 7. Logging and secret handling

Use explicitly constructed `log/slog` handlers. Stage 1 provides human-readable stderr logging at default `WARN`, plus an optional JSON Lines file destination at default `DEBUG`. Destination levels are independent. There is no default log file; its path is explicit. Use restrictive permissions where supported, external rotation, and fail initialization if a required destination cannot be opened. Apply structured redaction before fan-out, including known sensitive keys, nested attributes and secret-aware types.

Stage 3 adds extensible remote logging sinks and an optional built-in rotation capability. Resolved secret values must not be intentionally logged. Redaction cannot guarantee removal of secrets embedded in arbitrary free-form messages; application behavior and tests must prevent such messages.

## 8. Progress, errors and retries

Stage 1 defines a semantic progress interface and a plain stderr reporter. Stage 2 may render animated progress when terminal capability permits and must degrade safely when redirected or non-interactive. Stage 3 coordinates concurrent and nested progress so multiple operations do not corrupt terminal output or block application execution indefinitely.

Use classified errors, propagate cancellation, perform cleanup and retry only eligible infrastructure operations with bounded behavior. Do not retry whole application workflows automatically. Exact numeric exit-code mapping and retry backoff defaults remain implementation-level details to finalize while preserving these contracts.

## 9. Test and quality strategy

Stage 1 covers domain and application behavior, configuration precedence and validation, structured redaction, JSON output and stdout/stderr separation, cancellation, failures and exit behavior. Run `go vet`, `golangci-lint` and `go test -race`; exercise Linux and macOS cross-platform builds and smoke tests. Measure coverage without a mandatory percentage threshold.

Stage 2 adds golden output, terminal capability and animated-progress behavior tests, plus release checks. Stage 3 adds concurrent-progress stress tests, secret-provider integration tests and release artifact verification.

## 10. Developer experience and releases

Stage 1 provides a GitHub template, initialization script or documented setup, mise-managed tools and tasks, Prek hooks, CI aligned with local quality commands, and a README for setup, architecture and development. Initialization updates project names, Go module identity and configuration identifiers while leaving a runnable project. Hooks stay fast; full integration, race and release checks run separately.

Embed build/version metadata in Stage 1. Stage 2 uses GoReleaser and GitHub Actions for semantic Git-tagged GitHub Releases with archives, checksums and changelogs. Stage 3 adds SPDX JSON SBOMs, keyless Sigstore/Cosign signing of release checksums, GitHub provenance attestations and automated Homebrew tap publication. Artifact verification is tested. Sigstore signing does not replace Apple Developer ID signing or notarization.

## 11. Frozen stage boundaries

### Stage 1 — Foundation (mandatory)

Deliver a functional, testable and non-interactive-safe Go CLI in a generated repository: GitHub template with mise initialization tasks and Prek hooks; pragmatic hexagonal architecture and simulated infrastructure CLI; Cobra hierarchy, help, completions, flags, versions and standard command conventions; Koanf v2, TOML, strict layered configuration and source-relative paths; slog with stderr and optional JSONL file destinations and structured secret redaction; text/JSON results, stream separation, TTY detection and color controls; semantic progress interface with plain stderr reporter; classified errors, cancellation, cleanup and safe infrastructure retries; Linux/macOS AMD64/ARM64 builds and CI quality gates.

**Definition of done:** A newly instantiated project builds on all four targets, passes the agreed tests and quality gates, and runs the simulated resource commands end to end. JSON stays parseable with progress and logging enabled. Configuration precedence, secret masking, cancellation and exit behavior are covered by automated tests.

### Stage 2 — User experience and releases (mandatory)

Add Fang, improved help and error presentation; Lip Gloss styling and formatted output; Bubbles/Bubble Tea animated progress; terminal-aware degradation and enhanced console logging; per-leaf configuration provenance and `config show` / `config explain` with key filtering and masked text/JSON output; GoReleaser and semantic Git-tagged GitHub Releases with archives, checksums and changelogs; golden and terminal behavior tests.

**Definition of done:** Interactive progress works and redirected output is safe and deterministic. Configuration inspection identifies effective sources without revealing secrets. A tagged release publishes binaries for all four targets.

### Stage 3 — Advanced execution and security (mandatory)

Add concurrent and nested progress with coordinated terminal rendering; optional lazy 1Password CLI adapter behind a provider-independent interface; extensible remote logging sinks and optional built-in rotation; SPDX SBOMs, keyless Cosign checksum signing and GitHub provenance attestations; automated Homebrew tap publication; concurrency stress tests, integration tests and artifact verification.

**Definition of done:** Multiple simulated operations report progress without output corruption or indefinitely blocking application execution. Secret resolution stays on demand and out of inspection output. Release artifacts have verifiable signatures, provenance and SBOMs, and can be installed through Homebrew.

Every committed capability is implemented and tested in the template. External integrations remain opt-in at runtime: for example, `op`, remote log collectors and a Homebrew tap are not required merely to develop or run a generated application.

## 12. Explicit exclusions

The frozen scope excludes Windows; YAML/CSV output; a runtime plugin framework; a mandatory DI framework; configuration hot reload; automatic whole-workflow retries; multiple concrete secret-provider implementations; Debian/RPM repositories; byte-for-byte reproducible builds; macOS notarization; and a mandatory numerical coverage threshold.

## 13. Change control

ADR-001 through ADR-020 are the accepted baseline. Changes to the three stage scopes require an explicit ADR amendment. Implementation details may evolve without amendment when they preserve the accepted contracts, stage boundaries and definitions of done. The stages are sequential and independently verifiable. Stage 3 external integrations are runtime-optional.

**Specification version:** 1.0 — frozen.
