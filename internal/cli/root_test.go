package cli

import (
	"bytes"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func TestResourceListCommand(t *testing.T) {
	command := NewRootCommand(app.NewResourceService(memory.ResourceReader{}))
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
