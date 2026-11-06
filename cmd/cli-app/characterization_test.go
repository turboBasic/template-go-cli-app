package main

import (
	"strings"
	"testing"
)

const (
	characterizationVersion   = "1.2.3"
	characterizationCommit    = "abc1234"
	characterizationBuildDate = "2026-01-02T03:04:05Z"
)

// TestCharacterization pins the Stage 1 behavior of the built binary: exact bytes for
// completion, version, results and failures; help is asserted structurally because its
// bytes change when presentation does.
func TestCharacterization(t *testing.T) {
	binary := buildCLIProcess(t, "-X main.version="+characterizationVersion+
		" -X main.commit="+characterizationCommit+
		" -X main.buildDate="+characterizationBuildDate)

	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run("completion "+shell, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, "completion", shell)
			assertExit(t, code, 0)
			assertEmpty(t, "stderr", stderr)
			assertGolden(t, "characterization/completion-"+shell+".golden", []byte(stdout))
		})
	}

	t.Run("version", func(t *testing.T) {
		stdout, stderr, code := runCLIProcess(t, binary, "--version")
		assertExit(t, code, 0)
		assertEmpty(t, "stderr", stderr)
		assertGolden(t, "characterization/version.golden", []byte(stdout))
	})

	for _, test := range []struct {
		name string
		args []string
	}{
		{"resource-list-text", []string{"resource", "list"}},
		{"resource-list-json", []string{"resource", "list", "--output", "json"}},
		{"resource-status-text", []string{"resource", "status", "example-1"}},
		{"resource-status-json", []string{"resource", "status", "example-1", "--output", "json"}},
		{"resource-apply-text", []string{"resource", "apply", "example-1"}},
		{"resource-apply-json", []string{"resource", "apply", "example-1", "--output", "json"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, test.args...)
			assertExit(t, code, 0)
			assertGolden(t, "characterization/"+test.name+".stdout.golden", []byte(stdout))
			assertGolden(t, "characterization/"+test.name+".stderr.golden", []byte(stderr))
		})
	}

	for _, test := range []struct {
		name string
		args []string
	}{
		{"unknown-command", []string{"nope"}},
		{"unknown-flag", []string{"resource", "--bogus"}},
		{"missing-argument", []string{"resource", "apply"}},
		{"unknown-resource-json", []string{"resource", "apply", "invalid", "--output", "json"}},
		{"invalid-output", []string{"--output", "xml", "resource", "list"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, test.args...)
			assertExit(t, code, 2)
			assertEmpty(t, "stdout", stdout)
			assertGolden(t, "characterization/"+test.name+".stderr.golden", []byte(stderr))
		})
	}

	flags := []string{"--config", "--log-level", "--output", "--color", "--quiet", "--timeout"}
	for _, test := range []struct {
		name string
		args []string
		want []string
	}{
		{"help flag", []string{"--help"}, append([]string{"resource", "completion", "help"}, flags...)},
		{"help resource", []string{"help", "resource"}, append([]string{"list", "status", "apply"}, flags...)},
		{"help unknown topic", []string{"help", "nope"}, []string{"resource", "completion"}},
		{"resource status help", []string{"resource", "status", "--help"}, append([]string{"status"}, flags...)},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, code := runCLIProcess(t, binary, test.args...)
			assertExit(t, code, 0)
			assertEmpty(t, "stderr", stderr)
			for _, want := range test.want {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout does not mention %q:\n%s", want, stdout)
				}
			}
		})
	}
}

func assertExit(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("exit code = %d, want %d", got, want)
	}
}

func assertEmpty(t *testing.T, stream, got string) {
	t.Helper()
	if got != "" {
		t.Errorf("%s = %q, want empty", stream, got)
	}
}
