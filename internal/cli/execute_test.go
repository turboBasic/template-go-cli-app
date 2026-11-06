package cli

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
	"github.com/turboBasic/template-go-cli-app/internal/config"
)

func TestExecuteKeepsStage1RootFeatures(t *testing.T) {
	t.Run("completion is generated once", func(t *testing.T) {
		var want, got bytes.Buffer
		plain := newTestRootCommand(t)
		plain.SetOut(&want)
		plain.SetArgs([]string{"completion", "bash"})
		if err := plain.Execute(); err != nil {
			t.Fatal(err)
		}
		fanged := newTestRootCommand(t)
		fanged.SetOut(&got)
		fanged.SetArgs([]string{"completion", "bash"})
		if err := Execute(context.Background(), fanged); err != nil {
			t.Fatal(err)
		}
		if got.String() != want.String() || strings.Count(got.String(), "# bash completion V2 for cli-app") != 1 {
			t.Fatalf("completion bash through Execute differs from the Stage 1 script or repeats it")
		}
	})

	t.Run("man is an unknown command", func(t *testing.T) {
		command := newTestRootCommand(t)
		command.SetErr(&bytes.Buffer{})
		command.SetArgs([]string{"man"})
		if err := Execute(context.Background(), command); err == nil || !strings.Contains(err.Error(), `unknown command "man"`) {
			t.Fatalf("Execute(man) error = %v, want unknown command", err)
		}
	})

	t.Run("version keeps the Stage 1 template", func(t *testing.T) {
		var out bytes.Buffer
		command := newTestRootCommand(t)
		command.SetOut(&out)
		command.SetArgs([]string{"--version"})
		if err := Execute(context.Background(), command); err != nil {
			t.Fatal(err)
		}
		if want := "cli-app version 1.2.3 (commit abc123, built test-build)\n"; out.String() != want {
			t.Fatalf("--version = %q, want %q", out.String(), want)
		}
	})
}

func TestExecuteHelpAndCompletionSkipFactory(t *testing.T) {
	setupConfigEnvironment(t)
	for _, args := range [][]string{{"--help"}, {"help"}, {"completion", "bash"}, {"__complete", "resource", ""}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			factoryCalls := 0
			command := NewRootCommandWithFactory(func(context.Context, config.Config, *slog.Logger, app.ProgressReporter) (*app.ResourceService, func(context.Context) error, error) {
				factoryCalls++
				return nil, nil, errors.New("factory must not run")
			}, testBuildInfo(), nil)
			command.SetArgs(args)
			command.SetOut(&bytes.Buffer{})
			command.SetErr(&bytes.Buffer{})
			if err := Execute(context.Background(), command); err != nil {
				t.Fatalf("Execute(%v) error = %v", args, err)
			}
			if factoryCalls != 0 {
				t.Fatalf("factory calls = %d, want zero", factoryCalls)
			}
		})
	}
}

func TestExecuteReportsJoinedErrorOnce(t *testing.T) {
	cause := errors.New("interrupted by test")
	for _, test := range []struct {
		name    string
		run     func(*cobra.Command) error
		want    string
		wantErr error
	}{
		{"error and cause", func(*cobra.Command) error { return errors.New("operation failed") }, "operation failed\ninterrupted by test\n", cause},
		{"cause alone", func(*cobra.Command) error { return nil }, "interrupted by test\n", cause},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(cause)
			root := &cobra.Command{Use: "x", RunE: func(cmd *cobra.Command, _ []string) error { return test.run(cmd) }}
			var stderr bytes.Buffer
			root.SetErr(&stderr)
			root.SetArgs(nil)
			err := Execute(ctx, root)
			if !errors.Is(err, test.wantErr) || stderr.String() != test.want {
				t.Fatalf("Execute() = %v, stderr %q; want cause kept and stderr %q", err, stderr.String(), test.want)
			}
		})
	}
	t.Run("success prints nothing", func(t *testing.T) {
		root := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return nil }}
		var stderr bytes.Buffer
		root.SetErr(&stderr)
		root.SetArgs(nil)
		if err := Execute(context.Background(), root); err != nil || stderr.Len() != 0 {
			t.Fatalf("Execute() = %v, stderr %q; want nil and empty", err, stderr.String())
		}
	})
}
