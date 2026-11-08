# Go CLI App Template — Architecture Specification v1.0

**Status:** Approved
**Decision records:** [decisions/README.md](decisions/README.md)
**Target platforms:** Linux and macOS, AMD64 and ARM64
**Primary language:** Go

## 1. Purpose and objectives

Build a reusable GitHub template repository for modern Go command-line applications. Each use of the template creates an independent application; the template is not a shared runtime framework.

The template provides hierarchical commands, help, flags, shell completion and version reporting; separation of CLI presentation, application logic, domain logic and infrastructure; layered, strongly typed configuration; structured logging and secret redaction; text and JSON output; terminal-aware progress; cancellation and classified errors; automated quality checks; cross-platform releases; optional secret-provider integration; and release supply-chain verification.

## 2. Architectural principles

1. Use pragmatic hexagonal architecture: create boundaries where dependencies need to be replaceable or tested, without adding indirection without a use case.
2. Keep dependency direction inward. Domain logic does not depend on the CLI, presentation libraries or external-service implementations.
3. Construct and inject dependencies during application bootstrap.
4. Use typed Go contracts and classified errors rather than unstructured maps or implicit global state.
5. Keep command semantics and output contracts stable as presentation capabilities grow.
6. Make automation safe: results go to stdout; diagnostics, logs and progress go to stderr.
7. Treat secrets as sensitive values throughout configuration, logging and output.
8. Carry the minimum code and tests that demonstrate each contract, and keep external integrations runtime-optional.

## 3. Technology decisions

| Concern                          | Selected technology                                            |
| -------------------------------- | -------------------------------------------------------------- |
| CLI command framework            | Cobra                                                          |
| Rich CLI presentation            | Fang                                                           |
| Configuration                    | Koanf v2 with TOML                                             |
| Structured logging               | Go `log/slog` and custom handlers                              |
| Text and JSON output             | Go standard library and presentation adapters                  |
| Terminal capability detection    | Go terminal libraries                                          |
| Terminal styling                 | Lip Gloss                                                      |
| Animated progress                | Bubbles / Bubble Tea                                           |
| Concurrent progress coordination | Mutex-guarded progress reporters                               |
| Secret resolution                | Provider-independent interface; optional 1Password CLI adapter |
| Tool versions and task runner    | mise                                                           |
| Git hooks                        | Prek                                                           |
| Static analysis                  | `go vet`, `golangci-lint`                                      |
| CI                               | GitHub Actions                                                 |
| Release automation               | GoReleaser                                                     |
| Supply-chain verification        | Cosign, SPDX SBOM, GitHub attestations                         |
| Package distribution             | Automated Homebrew tap                                         |

Exact dependency versions are pinned during implementation and maintained under a dependency update policy; the baseline does not prescribe arbitrary version numbers.

## 4. Package boundaries

A reasonable layout is:

```text
cmd/cli-app/main.go
internal/cli/{root,resource,config}.go                  # Cobra commands and command wiring
internal/app/{resource,progress,secret}.go              # use cases, orchestration and the secret resolver interface
internal/domain/{resource}.go                           # models, invariants and business rules
internal/config/{config,load,provenance,validate}.go    # loading, source metadata and validation
internal/logging/{setup,redact,fanout}.go               # slog setup, redaction and destinations
internal/output/{renderer,terminal,progress}.go         # result rendering, terminal capabilities and progress
internal/infra/{memory,retry,onepassword}/              # external-facing adapters, simulated fixture and 1Password adapter
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

The loader retains the source metadata needed for path resolution. `config show [key]` is the single inspection command: it reports effective values and source identities, supports text and JSON output and key filtering, masks sensitive values, and never resolves secret references. Per-key origin tracking and source-relative path resolution may require a metadata-aware loading layer around Koanf.

## 6. CLI and output contract

Use hierarchical lowercase kebab-case commands, consistent global flags, help, shell completion and version information. Non-interactive use must not unexpectedly prompt. Results go to stdout; diagnostics and progress go to stderr. JSON output remains parseable under normal operation and failures. `text` and `json` output modes, terminal detection, color controls and safe redirected output are supported; Fang and Lip Gloss own presentation. `--color`, `CLI_APP_COLOR` and the config `color` key govern results and progress: `auto` styles a terminal unless `NO_COLOR` is non-empty, `always` styles regardless of `NO_COLOR`, `never` and JSON never style, and `CLICOLOR_FORCE` does not apply; help and errors follow terminal detection, `NO_COLOR` and `CLICOLOR_FORCE` as Fang does. Domain and application code remain independent of rendering libraries.

The reference application is a simulated infrastructure CLI and uses local fixtures without external credentials. Representative commands are `cli-app resource list`, `cli-app resource status` and `cli-app resource apply`. Commands that only demonstrate a template contract live under `cli-app demo`, show one contract each and are deleted by an application built from the template; `cli-app demo long-running` shows the determinate progress bar.

## 7. Logging and secret handling

Use explicitly constructed `log/slog` handlers. Logging provides human-readable stderr output at default `WARN`, plus an optional JSON Lines file destination at default `DEBUG`. Destination levels are independent. There is no default log file; its path is explicit. Use restrictive permissions where supported, external rotation, and fail initialization if a required destination cannot be opened. Apply structured redaction before fan-out, including known sensitive keys, nested attributes and secret-aware types.

Any additional destination attaches behind the same redaction, and rotation is external. Resolved secret values must not be intentionally logged. Redaction cannot guarantee removal of secrets embedded in arbitrary free-form messages; application behavior and tests must prevent such messages.

## 8. Progress, errors and retries

A semantic progress interface has a plain stderr reporter. Animated progress renders only when terminal capability permits and degrades safely when redirected or non-interactive. Where several operations report at once, rendering is coordinated so they do not corrupt terminal output.

Use classified errors, propagate cancellation, perform cleanup and retry only eligible infrastructure operations with bounded behavior. Do not retry whole application workflows automatically. Exact numeric exit-code mapping and retry backoff defaults remain implementation-level details to finalize while preserving these contracts.

## 9. Test and quality strategy

Tests cover domain and application behavior, configuration precedence and validation, structured redaction, JSON output and stdout/stderr separation, cancellation, failures and exit behavior. Run `go vet`, `golangci-lint` and `go test -race`; exercise Linux and macOS cross-platform builds and smoke tests. Measure coverage without a mandatory percentage threshold.

## 10. Developer experience and releases

The repository is a GitHub template with an initialization script or documented setup, mise-managed tools and tasks, Prek hooks, CI aligned with local quality commands, and a README for setup, architecture and development. Initialization updates project names, Go module identity and configuration identifiers while leaving a runnable project. Hooks stay fast; full integration, race and release checks run separately.

Build and version metadata is embedded. GoReleaser and GitHub Actions produce semantic Git-tagged GitHub Releases with archives, checksums and changelogs; releases may add SPDX JSON SBOMs, keyless Sigstore/Cosign signing of release checksums, GitHub provenance attestations and automated Homebrew tap publication. Sigstore signing does not replace Apple Developer ID signing or notarization.

## 11. Explicit exclusions

The scope excludes Windows; YAML/CSV output; a runtime plugin framework; a mandatory DI framework; configuration hot reload; automatic whole-workflow retries; multiple concrete secret-provider implementations; Debian/RPM repositories; byte-for-byte reproducible builds; macOS notarization; and a mandatory numerical coverage threshold.

## 12. Change control

The accepted decision records are the baseline. Delivery scope, order and definitions of done are tracked in the GitHub milestones and issues, not here. Implementation details may evolve without a new record when they preserve the accepted contracts; changing a contract requires a superseding ADR.

**Specification version:** 1.0
