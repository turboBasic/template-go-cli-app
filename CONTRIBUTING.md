# Contributing

## Setup

Tool versions are pinned in `mise.toml`; tasks are run with `just`.

```sh
mise trust       # first time in a fresh clone
mise install
```

## The loop

```sh
just fmt
just lint
just test
just build
```

## Questions and changes

Open a [GitHub issue](https://github.com/turboBasic/template-go-cli-app/issues) for a reproducible
bug or a change ready to be scoped. Use a pull request to discuss a proposed code change.

## Sending a change

- Use Conventional Commits; the PR title follows the same format.
- Update documentation when behavior or project conventions change.
- Read [`docs/ai-instructions.md`](docs/ai-instructions.md) and the relevant architecture decisions
  before changing project behavior.
