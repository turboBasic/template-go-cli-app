package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func TestResourceListCommand(t *testing.T) {
	command := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), testBuildInfo())
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
	for _, args := range [][]string{{"--help"}, {"help", "resource", "status"}, {"--version"}, {"completion", "bash"}, {"completion", "zsh"}, {"completion", "fish"}, {"completion", "powershell"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			command := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), testBuildInfo())
			command.SetArgs(args)
			var output bytes.Buffer
			command.SetOut(&output)
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if output.Len() == 0 {
				t.Fatal("command output is empty")
			}
			if args[0] == "--version" && (!strings.Contains(output.String(), "1.2.3") || !strings.Contains(output.String(), "abc123") || !strings.Contains(output.String(), "test-build")) {
				t.Fatalf("version output = %q", output.String())
			}
		})
	}
}

func TestResourceStatusAndApplyCommands(t *testing.T) {
	for _, tc := range []struct {
		args string
		want string
	}{
		{args: "status example-2", want: "example-2: pending\n"},
		{args: "apply example-2", want: "example-2: ready\n"},
	} {
		t.Run(tc.args, func(t *testing.T) {
			command := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), testBuildInfo())
			command.SetArgs(strings.Fields("resource " + tc.args))
			command.SetIn(bytes.NewBuffer(nil))
			var output bytes.Buffer
			command.SetOut(&output)
			if err := command.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if output.String() != tc.want {
				t.Fatalf("output = %q, want %q", output.String(), tc.want)
			}
		})
	}
}

func TestPersistentFlagsAndValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
		bad  bool
	}{
		{name: "global flags", args: []string{"--config", "local.toml", "--log-level", "debug", "--output", "json", "--color", "never", "--quiet", "resource", "list"}, want: "example-1"},
		{name: "flags after resource", args: []string{"resource", "--output", "text", "list"}, want: "example-1"},
		{name: "invalid output", args: []string{"--output", "yaml", "resource", "list"}, bad: true},
		{name: "invalid log level", args: []string{"--log-level", "trace", "resource", "list"}, bad: true},
		{name: "invalid color", args: []string{"--color", "sometimes", "resource", "list"}, bad: true},
		{name: "json with forced color", args: []string{"--output", "json", "--color", "always", "resource", "list"}, bad: true},
		{name: "missing name", args: []string{"resource", "status"}, bad: true},
		{name: "extra name", args: []string{"resource", "status", "example-1", "example-2"}, bad: true},
		{name: "unknown name", args: []string{"resource", "status", "missing"}, bad: true},
		{name: "extra list argument", args: []string{"resource", "list", "example-1"}, bad: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			command := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), testBuildInfo())
			command.SetArgs(tc.args)
			var output bytes.Buffer
			command.SetOut(&output)
			err := command.Execute()
			if tc.bad && err == nil {
				t.Fatal("Execute() error = nil, want validation error")
			}
			if !tc.bad && err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if tc.want != "" && !strings.Contains(output.String(), tc.want) {
				t.Fatalf("output = %q, want it to contain %q", output.String(), tc.want)
			}
		})
	}
}

func TestResourceNameCompletion(t *testing.T) {
	command := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), testBuildInfo())
	status, _, err := command.Find([]string{"resource", "status"})
	if err != nil {
		t.Fatal(err)
	}
	got, directive := status.ValidArgsFunction(status, nil, "example-2")
	if len(got) != 1 || got[0] != "example-2" || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion = (%v, %v), want [example-2] with no file completion", got, directive)
	}
}

func TestCommandTreesHaveIndependentFlags(t *testing.T) {
	first := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), testBuildInfo())
	second := NewRootCommand(app.NewResourceService(memory.NewResourceReader()), testBuildInfo())
	first.SetArgs([]string{"--output", "json", "resource", "list"})
	first.SetOut(&bytes.Buffer{})
	if err := first.Execute(); err != nil {
		t.Fatal(err)
	}
	second.SetArgs([]string{"resource", "list"})
	second.SetOut(&bytes.Buffer{})
	if err := second.Execute(); err != nil {
		t.Fatal(err)
	}
}

func testBuildInfo() BuildInfo {
	return BuildInfo{Version: "1.2.3", Commit: "abc123", BuildDate: "test-build"}
}
