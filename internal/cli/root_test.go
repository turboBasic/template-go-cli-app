package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func TestResourceListCommand(t *testing.T) {
	command := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), "test")
	command.SetArgs([]string{"resource", "list"})
	var output bytes.Buffer
	command.SetOut(&output)

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "NAME        STATUS\nexample-1   ready\nexample-2   pending\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRootFeatures(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}, {"completion", "bash"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			command := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), "1.2.3")
			command.SetArgs(args)
			var output bytes.Buffer
			command.SetOut(&output)
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if output.Len() == 0 {
				t.Fatal("command output is empty")
			}
			if args[0] == "--version" && !strings.Contains(output.String(), "1.2.3") {
				t.Fatalf("version output = %q", output.String())
			}
		})
	}
}
