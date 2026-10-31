package main

import (
	"fmt"
	"os"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

var version = "dev"

func main() {
	resources := memory.NewResourceReader()
	service := app.NewResourceService(resources)
	command := cli.NewRootCommand(service, version)
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
