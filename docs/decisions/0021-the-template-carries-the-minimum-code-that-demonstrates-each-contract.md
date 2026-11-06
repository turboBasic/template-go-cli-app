---
id: "0021"
status: accepted
date: 2026-11-06
scope: project
---

# ADR 0021 — The template carries the minimum code that demonstrates each contract

## Decision

The template ships the smallest code and tests that show each contract working. A capability enters the
template as a demonstration of a contract, not as a feature set to maintain. Delivery scope and order are
tracked in the issue tracker, not in a record.

## Context

ADR-0020 froze three stages and required every committed capability to be implemented and tested in the
template. A template is copied, so each extra file, golden or test matrix is something every user must
rewrite or delete: the 28 characterization golden files removed in #43 existed only to prove one migration.

## Options

### Carry the minimum that demonstrates each contract (SELECTED)

- Adopted because: users inherit patterns they can read in an afternoon and extend.
- Adopted despite: a contract is only as well pinned as its smallest test.

### Implement and test every committed capability (ADR-0020)

- Rejected because: it makes the template a maintained application, not an example of one.
- Rejected despite: nothing is left unproven when a user adopts a capability.

## Consequences

- Test scaffolding beyond what fails when a contract breaks, such as byte-for-byte output files or stress
  suites for a demo, is not added to the template.
- Each capability carries the smallest test that fails when its contract breaks.
- Reopen if a contract cannot be understood from the minimal example that demonstrates it.

## Links

No issue. Evidence: #43, #45. Supersedes [ADR 0020](0020-stage-definitions-of-done-and-scope-freeze.md).
