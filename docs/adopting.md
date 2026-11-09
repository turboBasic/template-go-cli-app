# Adopting the template command runtime

The template's `demo` group is an example application, not a dependency of the shared CLI root. Replace its registration with your own command group and typed settings while keeping shared help, completion, configuration, logging, output, progress, timeout, cancellation, and cleanup behavior.

## Register your application

In `cmd/cli-app/main.go`, construct the root with `cli.NewRootCommand(registration, buildInfo())`. Define a `cli.CommandRegistration` with:

- `Settings`: a struct of application defaults tagged with `koanf` keys and an optional validator. The loader combines this schema with shared settings, validates the union, and preserves precedence and provenance. Map command flags to config keys with `FlagValues`; choose a command-specific timeout with `OperationTimeout` when needed.
- `RegisterCommands`: add a static Cobra command tree. The callback receives `*cli.CommandRuntime`, which exposes the loaded shared and application settings, shared renderer, logger, and progress reporter.

Keep dependency construction in the selected command's `RunE` path. Cache a dependency in a group object when sibling commands share it. Call `runtime.RegisterCleanup` as soon as cleanup is known, including after partial setup, so the root can run it on setup errors too. The root runs registered cleanups once in reverse registration order, with a live context bounded by `timeouts.cleanup`; cleanup errors are joined with command errors.

Render an application-owned result with `runtime.Renderer.RenderResult(result, renderText)`. The text callback receives an `io.Writer` and `output.Options`; JSON uses the result's ordinary JSON encoding and skips the callback. Keep results on stdout and logs or progress on stderr. Resolve secrets only inside the operation that needs them.

## Remove the reference demo

Delete the demo registration and command implementation, then remove every demo-only dependency they leave unused:

1. In `cmd/cli-app/main.go`, replace `cli.NewDemoRegistration(...)` with your registration. Remove `cmd/cli-app/demo_factory.go` and its imports of the memory, 1Password, and retry adapters when your application does not use them.
2. Remove `internal/cli/demo_commands.go`, `demo_config.go`, `demo_output.go`, and `demo_registration.go`. Move the settings, flag mapping, timeout selection, command definitions, and text formatting that you need into your application-owned files. Keep `internal/cli/root.go`, `config.go`, and the shared lifecycle.
3. Remove demo-only keys from `.cli-app.example.toml`: `timeouts.demo.structured_output`, `timeouts.demo.not_found`, `timeouts.demo.secret_credential`, `retry.demo.flaky_dependency.{max_attempts,initial_delay,max_delay,jitter}`, and `demo.secret_credential.reference`. Their corresponding variables are `CLI_APP_TIMEOUTS_DEMO_STRUCTURED_OUTPUT`, `CLI_APP_TIMEOUTS_DEMO_NOT_FOUND`, `CLI_APP_TIMEOUTS_DEMO_SECRET_CREDENTIAL`, `CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_{MAX_ATTEMPTS,INITIAL_DELAY,MAX_DELAY,JITTER}`, and `CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE`. Remove their typed definitions, defaults, validation, `--max-attempts` flag mapping, and tests with the demo registration. Keys in TOML for removed settings remain errors. Environment variables for removed keys are ignored because unknown environment variables are ignored.
4. Remove or replace demo fixtures and adapters only when no retained command uses them: `internal/infra/memory/{resource,flaky}.go`, `internal/domain/resource.go`, `internal/app/{resource,flaky_dependency,long_running}.go`, and the 1Password resolver. Search their callers before deleting; the simulated resource fixture may be shared by multiple demos.
5. Update `README.md`, `docs/architecture-specification.md`, and any generated application docs that name `cli-app demo`, demo flags, demo settings, or demo fixtures. Update demo-specific tests; preserve tests for shared root behavior, output stream separation, strict unknown TOML keys, lazy secrets, cleanup and cancellation, and bounded read-only retries where the replacement application uses them. Keep ADR-0022's naming rule for reference demos unless that contract itself changes.

Keep retry construction in application composition and wrap only a safe read interface such as `app.StatusReader` with `internal/infra/retry`. Do not wrap a workflow or mutation. ADR-0022 continues to govern the names of reference demos that remain in the template.

## Check the replacement

Build and try the registered command in both output modes, inspect a shared setting, and confirm strict config behavior:

```sh
just build
./dist/cli-app --output text replacement
./dist/cli-app --output json replacement
./dist/cli-app config show output
./dist/cli-app --config removed-demo.toml replacement
./dist/cli-app --config unknown-app.toml replacement
```

The first command using a removed demo key and the command using an unknown application key must fail with configuration validation. A valid shared configuration must still load without the demo schema.
