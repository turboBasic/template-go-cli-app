package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

// DemoFactory creates the reference demo's service and its dependency cleanup.
type DemoFactory func(
	context.Context, config.Config, DemoConfig, *slog.Logger, app.ProgressReporter,
) (*app.ResourceService, func(context.Context) error, error)

// NewDemoRegistration returns the template's example command group as an application registration.
func NewDemoRegistration(factory DemoFactory, names []string) CommandRegistration {
	names = append([]string(nil), names...)
	return CommandRegistration{
		Settings:         DemoApplicationSettings(),
		FlagValues:       demoFlagValues,
		OperationTimeout: configuredOperationTimeout,
		RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
			group := &demoResourceGroup{runtime: runtime, factory: factory}
			root.AddCommand(newDemoCommand(group, names))
		},
	}
}

type demoResourceGroup struct {
	runtime     *CommandRuntime
	factory     DemoFactory
	runID       uint64
	initialized bool
	resources   *app.ResourceService
	err         error
}

func (g *demoResourceGroup) get(ctx context.Context) (*app.ResourceService, error) {
	if g.runID != g.runtime.RunID {
		g.runID = g.runtime.RunID
		g.initialized = false
		g.resources = nil
		g.err = nil
	}
	if !g.initialized {
		g.initialized = true
		if g.factory == nil {
			g.err = errors.New("demo resource factory is required")
		} else {
			resources, cleanup, err := g.factory(
				ctx,
				g.runtime.Configuration.Config,
				g.runtime.Configuration.Application.(DemoConfig),
				g.runtime.Logger,
				g.runtime.Progress,
			)
			g.runtime.RegisterCleanup(cleanup)
			if err != nil {
				g.err = fmt.Errorf("create application dependencies: %w", err)
			} else if resources == nil {
				g.err = errors.New("demo factory returned no service")
			} else {
				g.resources = resources
			}
		}
	}
	return g.resources, g.err
}

func configuredOperationTimeout(cmd *cobra.Command, application any) *time.Duration {
	settings, ok := application.(DemoConfig)
	if !ok || cmd.Parent() == nil || cmd.Parent().Name() != "demo" {
		return nil
	}
	switch strings.ReplaceAll(cmd.Name(), "-", "_") {
	case "structured_output":
		return settings.Timeouts.Demo.StructuredOutput
	case "not_found":
		return settings.Timeouts.Demo.NotFound
	case "secret_credential":
		return settings.Timeouts.Demo.SecretCredential
	default:
		return nil
	}
}
