package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	var settings DemoConfig
	return CommandRegistration{
		Settings:   DemoApplicationSettings(),
		FlagValues: demoFlagValues,
		PrepareApplication: func(value any) (any, error) {
			loaded, ok := value.(DemoConfig)
			if !ok {
				return nil, fmt.Errorf("expected demo settings, got %T", value)
			}
			settings = loaded
			return loaded, nil
		},
		RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
			group := &demoResourceGroup{runtime: runtime, factory: factory, settings: &settings}
			demo := newDemoCommand(group, names)
			runtime.SetCommandPolicy(demo, CommandPolicy{
				Setup: SetupPolicy{Declared: true, Run: group.setup},
			})
			root.AddCommand(demo)
		},
	}
}

type demoResourceGroup struct {
	runtime   *CommandRuntime
	factory   DemoFactory
	settings  *DemoConfig
	resources *app.ResourceService
}

func (g *demoResourceGroup) setup(cmd *cobra.Command, runtime *CommandRuntime) error {
	if g.factory == nil {
		return errors.New("demo resource factory is required")
	}
	resources, cleanup, err := g.factory(
		cmd.Context(),
		runtime.Configuration.Config,
		*g.settings,
		runtime.Logger,
		runtime.Progress,
	)
	runtime.RegisterCleanup(cleanup)
	if err != nil {
		return fmt.Errorf("create application dependencies: %w", err)
	}
	if resources == nil {
		return errors.New("demo factory returned no service")
	}
	g.resources = resources
	return nil
}
