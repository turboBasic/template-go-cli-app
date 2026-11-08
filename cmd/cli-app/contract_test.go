// cspell:ignore resourc compdef

package main

import (
	"strings"
	"testing"
)

// TestCLIContract runs the built binary through the exit-code and stream contract: usage
// errors exit 2 with a one-line diagnostic on stderr and nothing on stdout, and the Cobra-owned
// surfaces (completion, version, help) keep working.
func TestCLIContract(t *testing.T) {
	binary := buildCLIProcess(t, "-X main.version=1.2.3 -X main.commit=abc1234 -X main.buildDate=2026-01-02T03:04:05Z")

	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown command", []string{"nope"}, "unknown command \"nope\" for \"cli-app\"\n"},
		{"unknown command suggestion", []string{"resourc"}, "unknown command \"resourc\" for \"cli-app\"\n\nDid you mean this?\n\tresource\n\n"},
		{"unknown resource subcommand", []string{"resource", "bogus"}, "unknown command \"bogus\" for \"cli-app resource\"\n"},
		{"unknown completion subcommand", []string{"completion", "bogus"}, "unknown command \"bogus\" for \"cli-app completion\"\n"},
		{"extra argument to list", []string{"resource", "list", "extra"}, "unknown command \"extra\" for \"cli-app resource list\"\n"},
		{"extra argument to completion shell", []string{"completion", "bash", "extra"}, "unknown command \"extra\" for \"cli-app completion bash\"\n"},
		{"unknown flag", []string{"resource", "--bogus"}, "unknown flag: --bogus\n"},
		{"missing argument", []string{"resource", "apply"}, "accepts 1 arg(s), received 0\n"},
		{"unknown resource", []string{"resource", "apply", "invalid", "--output", "json"}, "unknown resource \"invalid\"; valid names are example-1, example-2\n"},
		{"invalid output", []string{"--output", "xml", "resource", "list"}, "decode effective configuration: invalid output \"xml\": must be text or json\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, test.args...)
			if code != 2 || stdout != "" || stderr != "validation error: "+test.want {
				t.Fatalf("%v = stdout %q, stderr %q, exit %d; want empty stdout, %q and exit 2", test.args, stdout, stderr, code, "validation error: "+test.want)
			}
		})
	}

	for shell, marker := range map[string]string{
		"bash":       "# bash completion V2 for cli-app",
		"zsh":        "#compdef cli-app",
		"fish":       "# fish completion for cli-app",
		"powershell": "# powershell completion for cli-app",
	} {
		t.Run("completion "+shell, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, "completion", shell)
			if code != 0 || stderr != "" || strings.Count(stdout, marker) != 1 {
				t.Fatalf("completion %s = exit %d, stderr %q, %d markers; want exit 0, empty stderr and one script", shell, code, stderr, strings.Count(stdout, marker))
			}
		})
	}

	t.Run("version", func(t *testing.T) {
		stdout, stderr, code := runCLIProcess(t, binary, "--version")
		if want := "cli-app version 1.2.3 (commit abc1234, built 2026-01-02T03:04:05Z)\n"; code != 0 || stderr != "" || stdout != want {
			t.Fatalf("--version = %q, stderr %q, exit %d; want %q", stdout, stderr, code, want)
		}
	})

	flags := []string{"--config", "--log-level", "--output", "--color", "--quiet", "--timeout"}
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"help flag", []string{"--help"}, append([]string{"resource", "completion", "help"}, flags...)},
		{"help resource", []string{"help", "resource"}, append([]string{"list", "status", "apply"}, flags...)},
		{"resource status help", []string{"resource", "status", "--help"}, append([]string{"status"}, flags...)},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, test.args...)
			if code != 0 || stderr != "" {
				t.Fatalf("%v = exit %d, stderr %q; want exit 0 and empty stderr", test.args, code, stderr)
			}
			for _, want := range test.want {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout does not mention %q:\n%s", want, stdout)
				}
			}
		})
	}

	for _, args := range [][]string{{"help", "nope"}, {"help", "resource", "nope"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, stderr, code := runCLIProcess(t, binary, args...)
			if code != 2 || !strings.Contains(stderr, `unknown command "nope"`) {
				t.Fatalf("%v = exit %d, stderr %q; want exit 2 and an unknown command diagnostic", args, code, stderr)
			}
		})
	}
}
