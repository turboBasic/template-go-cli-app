// cspell:ignore resourc compdef

package main

import (
	"strings"
	"testing"
)

// TestCLIContract runs the built binary through the exit-code and stream contract: usage
// errors exit 2 with a diagnostic naming the offending argument on stderr and nothing on stdout, and the Cobra-owned
// surfaces (completion, version, help) keep working.
func TestCLIContract(t *testing.T) {
	binary := buildCLIProcess(t, "-X main.version=1.2.3 -X main.commit=abc1234 -X main.buildDate=2026-01-02T03:04:05Z")

	// The tokens are the parts of a diagnostic a user acts on; the surrounding wording belongs to Cobra and
	// the template and may change without breaking the contract.
	for _, test := range []struct {
		name   string
		args   []string
		tokens []string
	}{
		{"unknown command", []string{"nope"}, []string{`"nope"`}},
		{"unknown command suggestion", []string{"dem"}, []string{`"dem"`, "demo"}},
		{"removed resource group", []string{"resource", "list"}, []string{`"resource"`}},
		{"unknown demo subcommand", []string{"demo", "bogus"}, []string{`"bogus"`}},
		{"extra argument to a demo", []string{"demo", "long-running", "extra"}, []string{`"extra"`}},
		{"non-positive timeout on a demo", []string{"demo", "long-running", "--timeout", "0s"}, []string{"--timeout"}},
		{"unknown completion subcommand", []string{"completion", "bogus"}, []string{`"bogus"`}},
		{"extra argument to structured-output", []string{"demo", "structured-output", "extra"}, []string{`"extra"`}},
		{"extra argument to completion shell", []string{"completion", "bash", "extra"}, []string{`"extra"`}},
		{"unknown flag", []string{"demo", "--bogus"}, []string{"--bogus"}},
		{"missing argument", []string{"demo", "secret-credential"}, []string{"arg"}},
		{"invalid output", []string{"--output", "xml", "demo", "structured-output"}, []string{"xml"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, test.args...)
			if code != 2 || stdout != "" {
				t.Fatalf("%v = stdout %q, stderr %q, exit %d; want empty stdout and exit 2", test.args, stdout, stderr, code)
			}
			for _, token := range test.tokens {
				if !strings.Contains(stderr, token) {
					t.Errorf("%v stderr = %q, want it to contain %q", test.args, stderr, token)
				}
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
		// The version tokens are the contract (scripts/check-release.sh reads "version <v>"); the rest of the sentence is not.
		tokens := []string{"version 1.2.3", "abc1234", "2026-01-02T03:04:05Z"}
		if code != 0 || stderr != "" || !containsAll(stdout, tokens...) {
			t.Fatalf("--version = %q, stderr %q, exit %d; want exit 0, empty stderr and the tokens %q", stdout, stderr, code, tokens)
		}
	})

	flags := []string{"--config", "--log-level", "--output", "--color", "--quiet", "--timeout"}
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"help flag", []string{"--help"}, append([]string{"demo", "completion", "help"}, flags...)},
		{"help demo", []string{"help", "demo"}, append([]string{"structured-output", "not-found", "secret-credential"}, flags...)},
		{"not-found help", []string{"demo", "not-found", "--help"}, append([]string{"not-found"}, flags...)},
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

	for _, args := range [][]string{{"help", "nope"}, {"help", "demo", "nope"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, stderr, code := runCLIProcess(t, binary, args...)
			if code != 2 || !strings.Contains(stderr, `unknown command "nope"`) {
				t.Fatalf("%v = exit %d, stderr %q; want exit 2 and an unknown command diagnostic", args, code, stderr)
			}
		})
	}
}
