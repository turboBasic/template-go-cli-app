# Replaceable command composition spike

Issue: [#107](https://github.com/turboBasic/template-go-cli-app/issues/107)
Base revision: `d261d25947fe2f7653c9717d28829bb1d5c87dcc` (`docs(architecture): define demo removal boundary`)
Run date: 2026-11-09

## Result

Prefer a registration function as the next design to refine. It adds one explicit seam where the
application supplies its command group and leaves the shared root responsible for configuration,
logging, output, progress, operation context, and final cleanup. The facade prototype performs the
same work but adds a wrapper that does not own any additional lifecycle in this tree.

This is a direction for #108, not an implementation-ready API. Both prototypes still expose
`*app.ResourceService` through `CommandRuntime`; they prove that the command group can be replaced,
but not that an adopter with unrelated application result types can replace the service contract.
The current renderer also accepts `domain.Resource` specifically. A genuinely application-owned
result shape needs an output boundary decision before #108 can claim that level of replacement.

## Prototypes and reproducibility

The throwaway checkouts are retained at:

- `/private/tmp/command-composition-spike/registration` — explicit command registrar.
- `/private/tmp/command-composition-spike/facade` — small `commandApp` facade.
- `/private/tmp/command-composition-spike/demo-free` — registration copy with the demo operation-timeout schema removed.

All three started from the base revision above. The registration and facade variants have the same
`resources list`, `resources apply`, and `resources check` group and the same root lifecycle changes.
The facade variant adds `cmd/cli-app/application.go`; `main.go` then calls `newCommandApp().root()`.
The registration variant has `main.go` pass `registerResourceCommands` to
`NewRootCommandWithFactory`. Both add a `CommandRuntime` carrying effective config, logger, progress,
the constructed resource service, and renderer. Both retain the root's once-only cleanup callback.
The fake secret resolver returns a fixed value and has no external side effects.

From either prototype directory, the build and focused root assertions used the following commands.
In this restricted environment, Go's build and module caches were redirected to writable temporary
directories; the module sources came from the pre-populated local module cache.

```sh
go build -o ./cli-app ./cmd/cli-app
go test ./internal/cli -run TestReplacementGroupUsesRootLifecycle -v
go test ./internal/cli
```

The build passed in both variants. The focused assertion passed in each, including four lifecycle
subtests and four no-initialization invocations; the full `internal/cli` package passed 147 tests in
each. The existing `cmd/cli-app` subprocess suite was also
run against the replacement binary. Its 32 failures were stale assertions for removed `demo`
commands and demo-only flags/help; the package's source compiled. Those assertions must be replaced
with the new command group's contract tests when the composition change is implemented.

The replacement binary was exercised in each variant with these commands (using
`/private/tmp/command-composition-spike/secret-reference.toml`, which sets a fake secret reference
and a three-attempt retry with 10ms/20ms delays):

```sh
./cli-app --config ../secret-reference.toml config show demo.secret_credential.reference
./cli-app --config ../secret-reference.toml resources apply example-1
./cli-app --config ../secret-reference.toml --output json resources list
./cli-app --log-level debug resources list
./cli-app --config ../secret-reference.toml resources check
./cli-app --config ../secret-reference.toml --timeout 1ms resources check
```

Text and JSON results were produced on stdout. Debug logging showed `listing resources` on stderr;
progress and retry events also went to stderr. The
successful apply resolved the fake credential but rendered only the resource. The timeout stopped
the retry wait. SIGINT cancellation was checked with:

```sh
bash -c './cli-app --config ../retry-slow.toml resources check >/private/tmp/command-composition-spike/<variant>/cancel.out 2>/private/tmp/command-composition-spike/<variant>/cancel.err & pid=$!; sleep 0.1; kill -INT "$pid"; rc=0; wait "$pid" || rc=$?; printf "exit=%s\n" "$rc"; cat /private/tmp/command-composition-spike/<variant>/cancel.err'
```

It exited 130 and cancelled the retry wait in both variants. The focused root
test also covers success, command failure, pre-cancelled context, timeout, and one cleanup call;
its help, completion, `help resources`, and `config show` cases assert zero factory calls. Existing
root tests additionally cover setup failure cleanup, bounded cleanup after cancellation, and
preserving command plus cleanup errors: `TestDependencyCleanupOnceAcrossCommandOutcomes`,
`TestExecuteCleansUpAfterNonDemoCommandFailure`, `TestDependencyCleanupGetsFreshBoundedContextAfterOperationCancellation`,
`TestHelpAndCompletionSkipConfigurationAndFactory`, and `TestConfigShowSkipsLoggingAndFactory`.

In the demo-free copy, removing `TimeoutConfig.Demo` and its example TOML section made
`timeouts.demo` unknown. From that copy, `./cli-app --config removed-demo-setting.toml resources list`
rejected the key, while `./cli-app --config shared-settings.toml config show log.level` displayed
the shared setting as `"info"`. The copy is deliberately limited to this schema slice: secret and
retry settings remain because the prototypes exercise those existing contracts.

## What the comparison shows

### Registration function

- **Authoring:** Build the Cobra group and pass it to the root constructor.
- **Dependencies:** Commands read the root-created `CommandRuntime`; one factory call supplies the
  shared service, logger, and reporter.
- **Cleanup:** The root owns progress shutdown, operation cancellation, dependency cleanup, and
  logger close. The factory supplies dependency cleanup.
- **Inspection:** Registration builds command definitions only; runtime is read in `RunE`, so
  help/completion/config inspection do not construct application dependencies.
- **Surface:** One registrar type and one root-constructor argument.
- **Limit:** `CommandRuntime.Resources` remains `*app.ResourceService`.

### Facade

- **Authoring:** Build the same group, then put factory, build info, names, and registration behind
  `commandApp.root()`.
- **Dependencies and inspection:** Same runtime and factory behavior; the facade does not change
  dependency sharing or help/config initialization.
- **Cleanup:** Same ownership. Moving cleanup into this facade would duplicate the root lifecycle.
- **Surface:** Registrar plus a facade type and method, with no new demonstrated capability.
- **Limit:** The wrapper hides `*app.ResourceService` from `main`, not from command authors.

The registration function is the smaller seam. A facade becomes useful only if it takes real ownership
of application composition or cleanup; this prototype did neither. Keep lifecycle ownership in the
root until a later design demonstrates a better boundary.

## Command-author checklist

Under either candidate, the author still has to:

- Register a command group and define its arguments, local flags, validation, help, and completion
  names.
- Map command flags to typed config overrides and provenance. The current `demoFlagValues` mapping
  is demo-specific; a replacement flag needs an equivalent explicit mapping.
- Add or move config fields, defaults, validation, environment-variable names, and the example TOML
  entries. Unknown-key validation must continue rejecting removed keys.
- Select the configured operation timeout for the command, or document that only `--timeout` applies.
  `configuredOperationTimeout` currently maps demo command names directly.
- Implement operation code in the application layer, classify errors, propagate the command context,
  and choose retries only for a safe dependency operation.
- Resolve credentials inside the operation that needs them, never during root setup or config
  inspection. Keep secret values out of output and logs.
- Render the result using a type the renderer understands, or provide a deliberate output adapter.
- Return application cleanup from the factory. The root still stops progress, cancels the operation,
  runs bounded dependency cleanup, closes logging, and joins cleanup errors with command errors.

The facade version has the same checklist. It centralizes a call site, not command-specific policy.

## Existing contracts and removal boundary

- `cmd/cli-app/main.go` currently creates the memory reader, supplies its names, and chooses the
  demo-specific factory. A registration design keeps the root invocation visible here. A facade can
  hide those arguments, but should not hide the fact that command-specific policy still exists.
- `cmd/cli-app/demo_factory.go` builds the resource service, attaches the lazy secret resolver, and
  opts the flaky status reader into bounded retry. Preserve the narrow retry target and cleanup
  callback shape if the factory moves.
- `internal/cli/root.go` owns config loading before dependency construction, root flags, renderer,
  logger, progress, operation context, and teardown. It also currently creates the demo group and
  special-cases demo config flags/timeouts. Replace those demo-specific hooks with the chosen
  registration seam; do not move shared lifecycle behavior into each command.
- `internal/cli/demo_config.go` contains `--max-attempts` config mapping and demo-name timeout
  routing. Both need a command-owned replacement or removal.
- `internal/config/` owns demo secret, retry, and timeout schema, defaults, validation, env overlays,
  provenance, and tests. Remove only settings that are truly unused after the replacement is known.
  In the demo-free copy, `timeouts.demo.*` and `CLI_APP_TIMEOUTS_DEMO_*` disappear with the schema;
  `[log]` remains accepted. Retry and secret settings are still used by this prototype, so removing
  them would break demonstrated behavior.
- `internal/output/renderer.go` renders `domain.Resource` and config entries. No renderer change was
  needed for this prototype. It does not yet support arbitrary application result types without an
  adapter or a broader output contract.
- The flaky reader is a deterministic local fixture in `internal/infra/memory/flaky.go`; it can be
  deleted if no replacement command uses retries. The ordinary resource reader remains needed by
  the reference application. Remove demo-only subprocess/help tests and update config fixtures,
  `.cli-app.example.toml`, README command examples, and ADR/spec text with the same migration.

## Specification and ADR changes

If #108 adopts registration as a supported replacement boundary:

1. Update Architecture Specification section 4 to name the root registration seam and distinguish
   shared lifecycle from application command registration.
2. Update section 6 to describe the reference command group as a removable example and make command
   result rendering's supported type boundary explicit.
3. Supersede ADR-0022 only if its accepted rule that reference commands are demos is being changed.
   If the ADR remains a statement about the reference application's current examples, no ADR is
   needed for the internal registration implementation. ADR-0021 remains unchanged.

Do not write an ADR before #108 resolves whether the replacement boundary is a lasting architecture
contract or just a template implementation detail. The report itself does not change production
composition.

## Limitations and follow-up

The replacement currently reuses `ResourceService` and `domain.Resource`, so it is a real non-demo
command group through the root but not proof that an unrelated application can supply arbitrary
services and result types. The focused lifecycle assertion uses deterministic in-memory resources
and a fake secret resolver; retry uses short real timers rather than a fake clock. The full CLI
subprocess suite still encodes demo command behavior and fails until those tests are migrated. These
limits should be part of #108's acceptance criteria, not reasons to weaken the existing cleanup,
configuration, retry, or secret guarantees.
