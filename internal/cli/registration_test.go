package cli

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/config"
	"github.com/turboBasic/template-go-cli-app/internal/output"
)

type replacementResult struct {
	Count int `json:"count"`
}

type replacementService struct{}

func (replacementService) Result() replacementResult { return replacementResult{Count: 4} }

type replacementSettings struct {
	Label string `koanf:"label"`
}

func TestReplacementRegistrationUsesSharedTextAndJSONRenderer(t *testing.T) {
	setupConfigEnvironment(t)
	for _, test := range []struct {
		name, format, want string
		wantTextCalls      int
	}{
		{name: "text callback", format: "text", want: "count: 4\n", wantTextCalls: 1},
		{name: "normal JSON", format: "json", want: "{\"count\":4}\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			textCalls := 0
			registration := CommandRegistration{
				Settings: config.ApplicationSettings{Defaults: replacementSettings{Label: "sample"}},
				RegisterCommands: func(root *cobra.Command, runtime *CommandRuntime) {
					root.AddCommand(&cobra.Command{Use: "replacement", RunE: func(_ *cobra.Command, _ []string) error {
						service := replacementService{}
						result := service.Result()
						if runtime.Configuration.Application.(replacementSettings).Label != "sample" || runtime.Logger == nil || runtime.Progress == nil {
							return fmt.Errorf("shared runtime or registered config was not populated")
						}
						return runtime.Renderer.RenderResult(result, func(w io.Writer, options output.Options) error {
							textCalls++
							if options.Format != output.FormatText {
								return fmt.Errorf("text options format = %q", options.Format)
							}
							_, err := fmt.Fprintf(w, "count: %d\n", result.Count)
							return err
						})
					}})
				},
			}
			root := NewRootCommand(registration, BuildInfo{})
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"--output", test.format, "replacement"})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != test.want {
				t.Fatalf("stdout = %q, want %q", stdout.String(), test.want)
			}
			if textCalls != test.wantTextCalls {
				t.Fatalf("text callback calls = %d, want %d", textCalls, test.wantTextCalls)
			}
		})
	}
}
