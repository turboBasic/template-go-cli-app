# Security policy

## Scope

This repository is a template for a Go command-line application. Its current security-relevant
contents are the starter CLI in `cmd/cli-app`, Go module and tool dependencies, and GitHub Actions
workflows used to check changes. It does not currently publish a hosted service or a release binary.

Please report vulnerabilities that could affect people adopting this template, including:

- A security defect in starter code that could carry into an application created from the template.
- A vulnerable dependency or unsafe build/tooling configuration included in the template.
- A GitHub Actions issue that could expose repository data, credentials, or write access beyond the
  permissions the workflow needs.

This policy does not promise support for downstream applications generated from the template. Their
owners are responsible for reviewing and updating their own copies.

## Reporting

Report a suspected vulnerability through [GitHub private vulnerability reporting][private-reporting].
Do not open a public issue for an exploitable problem. Include the affected file, dependency, or
workflow; the impact; and steps to reproduce when safe to share. Do not include live credentials or
other secrets in the report.

If private vulnerability reporting is unavailable, contact [@turboBasic][turbobasic] privately on
GitHub and ask for a secure reporting route. Please do not send sensitive details in a public issue.

The maintainer will review reports as project work allows. There is no guaranteed response or fix
timeline.

## Secrets and CI

Do not commit credentials, including in examples or test fixtures. The current CI workflow requests
read-only `contents` permission and calls reusable workflows from `turboBasic/github-actions`. A
report about the reusable workflow itself may belong in that repository's
[security reporting channel][sibling-reporting]; report here if this repository's call site,
permissions, or configuration creates the vulnerability.

## Supported versions

There are no separately supported release versions at this time. Reports against the current default
branch are in scope.

[private-reporting]: https://github.com/turboBasic/template-go-cli-app/security/advisories/new
[turbobasic]: https://github.com/turboBasic
[sibling-reporting]: https://github.com/turboBasic/github-actions/security/advisories/new
