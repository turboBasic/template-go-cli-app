# Architecture decisions

Accepted decisions for the [architecture specification][architecture-specification]. A decision changes only through a superseding record.

| ADR                                                                                         | Status     |
| ------------------------------------------------------------------------------------------- | ---------- |
| [ADR-0001: Project strategy][ADR-0001]                                                      | accepted   |
| [ADR-0002: Architecture style][ADR-0002]                                                    | accepted   |
| [ADR-0003: CLI framework progression][ADR-0003]                                             | accepted   |
| [ADR-0004: Standard CLI conventions][ADR-0004]                                              | accepted   |
| [ADR-0005: Configuration framework and format][ADR-0005]                                    | accepted   |
| [ADR-0006: Configuration discovery and precedence][ADR-0006]                                | accepted   |
| [ADR-0007: Strict configuration and path resolution][ADR-0007]                              | accepted   |
| [ADR-0008: Local logging defaults and destinations][ADR-0008]                               | accepted   |
| [ADR-0009: Logging fan-out, redaction and extension][ADR-0009]                              | accepted   |
| [ADR-0010: Output formats and terminal awareness][ADR-0010]                                 | accepted   |
| [ADR-0011: Progress reporting progression][ADR-0011]                                        | accepted   |
| [ADR-0012: Errors, cancellation, cleanup and retries][ADR-0012]                             | accepted   |
| [ADR-0014: Versioning and release automation][ADR-0014]                                     | accepted   |
| [ADR-0015: Template setup and developer tooling][ADR-0015]                                  | accepted   |
| [ADR-0016: Reference application][ADR-0016]                                                 | superseded |
| [ADR-0017: Configuration provenance and inspection][ADR-0017]                               | accepted   |
| [ADR-0018: Secret provider and resolution][ADR-0018]                                        | accepted   |
| [ADR-0019: Supply-chain security and distribution][ADR-0019]                                | accepted   |
| [ADR-0020: Stage definitions of done and scope freeze][ADR-0020]                            | superseded |
| [ADR-0021: The template carries the minimum code that demonstrates each contract][ADR-0021] | accepted   |
| [ADR-0022: Reference commands are demos named for the contract they demonstrate][ADR-0022]  | accepted   |
| [ADR-0023: Application commands register behind the shared CLI runtime][ADR-0023]           | accepted   |

## Source and rationale note

This export is grounded in the retrieved final specification, accepted ADR register and final stage-freeze decision. The retrieved conversation record did not expose an older-page cursor or the original full text of ADR-001 through ADR-017. Accordingly, these individual records preserve the accepted decisions and recorded constraints without inventing missing decision-specific rationale or alternatives.

[architecture-specification]: ../architecture-specification.md
[ADR-0001]: 0001-project-strategy.md
[ADR-0002]: 0002-architecture-style.md
[ADR-0003]: 0003-cli-framework-progression.md
[ADR-0004]: 0004-standard-cli-conventions.md
[ADR-0005]: 0005-configuration-framework-and-format.md
[ADR-0006]: 0006-configuration-discovery-and-precedence.md
[ADR-0007]: 0007-strict-configuration-and-path-resolution.md
[ADR-0008]: 0008-local-logging-defaults-and-destinations.md
[ADR-0009]: 0009-logging-fan-out-redaction-and-extension.md
[ADR-0010]: 0010-output-formats-and-terminal-awareness.md
[ADR-0011]: 0011-progress-reporting-progression.md
[ADR-0012]: 0012-errors-cancellation-cleanup-and-retries.md
[ADR-0014]: 0014-versioning-and-release-automation.md
[ADR-0015]: 0015-template-setup-and-developer-tooling.md
[ADR-0016]: 0016-reference-application.md
[ADR-0017]: 0017-configuration-provenance-and-inspection.md
[ADR-0018]: 0018-secret-provider-and-resolution.md
[ADR-0019]: 0019-supply-chain-security-and-distribution.md
[ADR-0020]: 0020-stage-definitions-of-done-and-scope-freeze.md
[ADR-0021]: 0021-the-template-carries-the-minimum-code-that-demonstrates-each-contract.md
[ADR-0022]: 0022-reference-commands-are-demos-named-for-their-contract.md
[ADR-0023]: 0023-replaceable-command-composition.md
