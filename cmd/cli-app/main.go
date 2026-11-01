package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

var version = "dev"
var commit = "unknown"
var buildDate = "unknown"

var (
	errInterruptSignal = errors.New("interrupted by SIGINT")
	errTerminateSignal = errors.New("terminated by SIGTERM")
)

func main() {
	resources := memory.NewResourceReader()
	command := cli.NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger) (*app.ResourceService, func(context.Context) error, error) {
		return app.NewResourceService(resources, logger), nil, nil
	}, buildInfo(), resources.Names())
	os.Exit(executeWithSignals(func(ctx context.Context) error {
		command.SetContext(ctx)
		return runCommand(command)
	}, os.Stderr))
}

func runCommand(command *cobra.Command) error {
	selected, err := command.ExecuteC()
	if err != nil && selected != nil && !selected.Runnable() {
		return app.Classify(app.ErrValidation, err)
	}
	return err
}

func execute(run func() error, stderr io.Writer) int {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return exitCode(err)
	}
	return 0
}

func executeWithSignals(run func(context.Context) error, stderr io.Writer) int {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(context.Canceled)
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	result := make(chan int, 1)
	go func() {
		result <- execute(func() error {
			err := run(ctx)
			if cause := context.Cause(ctx); cause != nil {
				err = errors.Join(err, cause)
			}
			return err
		}, stderr)
	}()
	select {
	case code := <-result:
		return code
	case first := <-signals:
		cancel(signalCause(first))
		select {
		case code := <-result:
			return code
		case second := <-signals:
			return exitCode(signalCause(second))
		}
	}
}

func exitCode(err error) int {
	switch {
	case errors.Is(err, errInterruptSignal):
		return 130
	case errors.Is(err, errTerminateSignal):
		return 143
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, app.ErrTimeout):
		return 124
	case errors.Is(err, app.ErrValidation):
		return 2
	default:
		return 1
	}
}

func signalCause(signal os.Signal) error {
	switch signal {
	case os.Interrupt:
		return errInterruptSignal
	case syscall.SIGTERM:
		return errTerminateSignal
	default:
		return fmt.Errorf("unexpected termination signal %s", signal)
	}
}

func buildInfo() cli.BuildInfo {
	build := cli.BuildInfo{Version: version, Commit: commit, BuildDate: buildDate}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return build
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if build.Commit == "unknown" {
				build.Commit = setting.Value
			}
		case "vcs.time":
			if build.BuildDate == "unknown" {
				build.BuildDate = setting.Value
			}
		}
	}
	return build
}
