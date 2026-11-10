# CLI architecture references

- [Go module layout](https://go.dev/doc/modules/layout) recommends starting simply and describes `internal` and `cmd` layouts.
- [Command Line Interface Guidelines](https://clig.dev/) covers output streams, progress, automation, and human-facing behavior.
- [GitHub CLI](https://github.com/cli/cli) is a production Go CLI to study; its [extension documentation](https://docs.github.com/en/github-cli/github-cli/creating-github-cli-extensions) describes executable extensions.
- [Bubble Tea examples](https://github.com/charmbracelet/bubbletea/tree/main/examples) demonstrate progress and terminal interaction; examples are not complete production architectures.
- [Structured logging with slog](https://go.dev/blog/slog) describes Go's structured logging and value redaction.
- [Google Cloud retry guidance](https://docs.cloud.google.com/storage/docs/retry-strategy) discusses retryable responses, idempotency, backoff, and jitter. [Handling Overload](https://sre.google/sre-book/handling-overload/) discusses retry budgets and avoiding retries at multiple layers.
- [Go plugin documentation](https://pkg.go.dev/plugin) describes platform and build compatibility limitations.

These sources cover complementary concerns, rather than prescribing one complete CLI architecture. Compare them against the [architecture specification](../architecture-specification.md) before changing the template.
