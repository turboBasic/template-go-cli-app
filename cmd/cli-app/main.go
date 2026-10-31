package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

var version = "dev"
var commit = "unknown"
var buildDate = "unknown"

func main() {
	resources := memory.NewResourceReader()
	service := app.NewResourceService(resources)
	command := cli.NewRootCommand(service, buildInfo(), resources.Names())
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
