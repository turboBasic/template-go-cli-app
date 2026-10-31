# Go CLI App Template · 🦫

[![CI][ci-badge]][ci] [![Go 1.27.3][go-badge]][go] [![License: MIT][license-badge]][license]

A GitHub template for starting Go command-line applications. It includes a pinned Go toolchain,
development tasks, quality checks, and an architecture specification for the application as it grows.

## Get started

Create a repository with GitHub's **Use this template** button, then clone it and run:

```sh
mise trust       # first time in a fresh clone
mise install
mise run init
just build
```

Initialization prompts for the project name, Go module path (including your GitHub account or
organization), and configuration/command identifier. It updates the starter command, configuration
paths and README title in the generated repository.

For scripted setup, pass all three values in order:

```sh
mise run init "Example App" github.com/example/example-app example-app
```

## Development

Run project tasks with `just`:

```sh
just fmt
just lint
just test
just build
```

See the [architecture specification][architecture] for the planned application structure and
capabilities. [Contributing][contributing] covers setup and pull requests.

## License

[MIT][license].

[architecture]: docs/architecture-specification.md
[ci]: https://github.com/turboBasic/template-go-cli-app/actions/workflows/ci.yml?query=branch%3Amain
[ci-badge]: https://github.com/turboBasic/template-go-cli-app/actions/workflows/ci.yml/badge.svg?branch=main
[contributing]: CONTRIBUTING.md
[go]: https://go.dev/
[go-badge]: https://img.shields.io/badge/Go-1.27.3-00ADD8.svg
[license]: LICENSE
[license-badge]: https://img.shields.io/badge/license-MIT-blue.svg
