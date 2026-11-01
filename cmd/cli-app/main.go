package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

var version = "dev"
var commit = "unknown"
var buildDate = "unknown"

func main() {
	resources := memory.NewResourceReader()
	command := cli.NewRootCommandWithFactory(func(_ context.Context, _ config.Config, logger *slog.Logger) (*app.ResourceService, error) {
		return app.NewResourceService(resources, logger), nil
	}, buildInfo(), resources.Names())
	os.Exit(execute(command.Execute, os.Stderr))
}

func execute(run func() error, stderr io.Writer) int {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
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
