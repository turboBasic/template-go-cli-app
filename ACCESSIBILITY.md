# Accessibility

This project provides command-line tools and documentation. People should be able to use the tools
and understand their output without relying on visual formatting alone. This page describes current
priorities, known limitations, and how to report an accessibility barrier.

## Priorities

- Keep instructions and important results available as text.
- Preserve meaningful labels and readable output in terminals.
- Keep repository documentation and contribution paths clear and usable.

For any web-facing changes, WCAG 2.2 Level AA is an aspirational target, not a claim of conformance.
No accessibility evaluation has been completed. [WCAG 2.2](https://www.w3.org/TR/WCAG22/) is the
current W3C Recommendation.

## Contributor expectations

- Do not convey information through color or visual formatting alone.
- For changes to public output or interfaces, describe what you checked and name the operating
  system, terminal, and assistive technology when applicable. Screenshots or recordings are optional.
- The project has no dedicated automated accessibility checks or published test matrix. Passing
  existing checks does not establish accessibility conformance.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the contribution process.

## Reporting accessibility issues

Report a barrier in a [GitHub issue](https://github.com/turboBasic/template-go-cli-app/issues/new).
Include the affected command or page, what happened, and what you expected. Operating system,
terminal, and assistive technology details can help reproduce it. You do not need to disclose a
disability, and screenshots or recordings are optional.

There is no response-time commitment. The maintainer reviews reports alongside other project work
and may ask for clarification or post a workaround in the issue.

## Supported environments and known limitations

No terminal, operating system, screen reader, or other assistive technology combination has been
systematically evaluated or is claimed as supported for accessibility. These are the limitations
currently documented; the absence of other reports does not establish that no other barriers exist.

Use a [GitHub issue](https://github.com/turboBasic/template-go-cli-app/issues/new) to suggest an
improvement to this statement or the project's accessibility practices.
