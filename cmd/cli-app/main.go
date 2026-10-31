package main

import (
	"fmt"
	"os"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func main() {
	resources := memory.ResourceReader{}
	service := app.NewResourceService(resources)
	command := cli.NewRootCommand(service)
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
