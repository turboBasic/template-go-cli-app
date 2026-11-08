package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFullPrecedence(t *testing.T) {
	root := t.TempDir()
	user := filepath.Join(root, "user.toml")
	project := filepath.Join(root, "project.toml")
	writeFile(t, user, "output = 'text'\n[log]\nlevel = 'info'\n")
	writeFile(t, project, "[log]\nlevel = 'error'\n")

	got, err := load(ConfigFiles{User: user, Project: project}, []string{
		"CLI_APP_LOG_LEVEL=warn",
		"CLI_APP_QUIET=true",
		"CLI_APP_UNKNOWN=ignored",
	}, map[string]FlagValue{
		"log-level": {Value: "debug", Changed: true, Source: "--log-level"},
		"output":    {Value: "json", Changed: true, Source: "--output"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Log: LogConfig{Level: "debug", File: FileLogConfig{Level: "debug"}}, Output: "json", Color: "auto", Quiet: true, Timeouts: TimeoutConfig{Cleanup: 5 * time.Second}, Retry: Defaults().Retry}
	if got.Config != want {
		t.Fatalf("Config = %+v, want %+v", got.Config, want)
	}
	for key, wantSource := range map[string]Source{
		"log.level": "--log-level",
		"output":    "--output",
		"quiet":     "CLI_APP_QUIET",
		"color":     defaultSource,
	} {
		if got.Sources[key] != wantSource {
			t.Errorf("source for %q = %q, want %q", key, got.Sources[key], wantSource)
		}
	}
}

func TestLoadUnchangedFlagLeavesEnvironmentEffective(t *testing.T) {
	got, err := load(ConfigFiles{}, []string{"CLI_APP_LOG_LEVEL=info"}, map[string]FlagValue{
		"log-level": {Value: "warn", Changed: false, Source: "--log-level"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Config.Log.Level != "info" || got.Sources["log.level"] != "CLI_APP_LOG_LEVEL" {
		t.Fatalf("effective console log level/source = %q/%q, want info/CLI_APP_LOG_LEVEL", got.Config.Log.Level, got.Sources["log.level"])
	}
}

func TestLoadInvalidEnvironmentTypeNamesVariable(t *testing.T) {
	_, err := load(ConfigFiles{}, []string{"CLI_APP_QUIET=sometimes"}, nil)
	if err == nil || !strings.Contains(err.Error(), "CLI_APP_QUIET") || !strings.Contains(err.Error(), "invalid boolean") {
		t.Fatalf("load() error = %v, want invalid CLI_APP_QUIET error", err)
	}
}

func TestLoadNestedLoggingEnvironment(t *testing.T) {
	got, err := load(ConfigFiles{}, []string{
		"CLI_APP_LOG_LEVEL=error",
		"CLI_APP_LOG_FILE_LEVEL=info",
		"CLI_APP_LOG_FILE_PATH=out/log.jsonl",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Config.Log.Level != "error" || got.Config.Log.File.Level != "info" {
		t.Fatalf("logging levels = %q/%q, want error/info", got.Config.Log.Level, got.Config.Log.File.Level)
	}
	if filepath.Base(got.Config.Log.File.Path) != "log.jsonl" || got.Sources["log.file.path"] != "CLI_APP_LOG_FILE_PATH" {
		t.Fatalf("file path/source = %q/%q, want working-directory path from CLI_APP_LOG_FILE_PATH", got.Config.Log.File.Path, got.Sources["log.file.path"])
	}
}

func TestLoadOperationTimeoutEnvironment(t *testing.T) {
	got, err := load(ConfigFiles{}, []string{
		"CLI_APP_TIMEOUTS_DEMO_NOT_FOUND=90s",
		"CLI_APP_TIMEOUTS_CLEANUP=3s",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Config.Timeouts.Demo.NotFound == nil || *got.Config.Timeouts.Demo.NotFound != 90*time.Second || got.Config.Timeouts.Cleanup != 3*time.Second {
		t.Fatalf("timeouts = %+v, want status 90s and cleanup 3s", got.Config.Timeouts)
	}
}

func TestLoadStatusRetryPolicyFromEnvironment(t *testing.T) {
	got, err := load(ConfigFiles{}, []string{
		"CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_MAX_ATTEMPTS=4",
		"CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_INITIAL_DELAY=25ms",
		"CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_MAX_DELAY=2s",
		"CLI_APP_RETRY_DEMO_FLAKY_DEPENDENCY_JITTER=0.3",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := RetryPolicyConfig{MaxAttempts: 4, InitialDelay: 25 * time.Millisecond, MaxDelay: 2 * time.Second, Jitter: 0.3}
	if got.Config.Retry.Demo.FlakyDependency != want {
		t.Fatalf("status retry policy = %+v, want %+v", got.Config.Retry.Demo.FlakyDependency, want)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCredentialFromFileAndEnvironment(t *testing.T) {
	userPath := filepath.Join(t.TempDir(), "user.toml")
	if err := os.WriteFile(userPath, []byte("[demo.secret_credential]\nreference = 'op://file/item/field'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		env        []string
		want, from string
	}{
		{name: "file", want: "op://file/item/field", from: userPath},
		{name: "environment", env: []string{"CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE=op://env/item/field"}, want: "op://env/item/field", from: "CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE"},
		{name: "empty environment clears", env: []string{"CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE="}, want: "", from: "CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := load(ConfigFiles{User: userPath}, test.env, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Config.Demo.SecretCredential.Reference != test.want || got.Sources["demo.secret_credential.reference"] != Source(test.from) {
				t.Fatalf("credential/source = %q/%q, want %q/%q", got.Config.Demo.SecretCredential.Reference, got.Sources["demo.secret_credential.reference"], test.want, test.from)
			}
		})
	}
	if _, err := load(ConfigFiles{}, []string{"CLI_APP_DEMO_SECRET_CREDENTIAL_REFERENCE=hunter2"}, nil); err == nil || !strings.Contains(err.Error(), "demo.secret_credential.reference") {
		t.Fatalf("load() error = %v, want a demo.secret_credential.reference validation error", err)
	}
}

func TestLoadRejectsFlagWithoutKey(t *testing.T) {
	_, err := load(ConfigFiles{}, nil, map[string]FlagValue{"no-such-flag": {Value: "debug", Source: "--no-such-flag"}})
	if err == nil || !strings.Contains(err.Error(), "--no-such-flag") {
		t.Fatalf("load() error = %v, want an error naming --no-such-flag", err)
	}
}
