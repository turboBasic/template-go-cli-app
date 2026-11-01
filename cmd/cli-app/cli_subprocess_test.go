package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIProcessSeparatesJSONResultsAndDiagnostics(t *testing.T) {
	binary := buildCLIProcess(t)
	stdout, stderr, code := runCLIProcess(t, binary, "resource", "apply", "invalid", "--output", "json")
	if stdout != "" || code != 2 || !strings.Contains(stderr, "unknown resource") {
		t.Fatalf("invalid JSON command = stdout %q, stderr %q, exit %d; want empty stdout, diagnostic stderr, exit 2", stdout, stderr, code)
	}

	stdout, stderr, code = runCLIProcess(t, binary, "resource", "status", "example-1", "--output", "json")
	var result struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || result != (struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}{Name: "example-1", Status: "ready"}) {
		t.Fatalf("successful JSON = %q (decode error %v), want the existing resource result shape", stdout, err)
	}
	if code != 0 || stderr != "" {
		t.Fatalf("successful JSON command = stderr %q, exit %d; want empty stderr and exit 0", stderr, code)
	}

	stdout, stderr, code = runCLIProcess(t, binary, "resource", "apply", "example-1", "--output", "json")
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || result != (struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}{Name: "example-1", Status: "ready"}) {
		t.Fatalf("applied JSON = %q (decode error %v), want the existing resource result shape", stdout, err)
	}
	if code != 0 || stderr != "applying: Applying resource\nvalidating: Validating resource\ncomplete: Resource applied (1/1)\n" {
		t.Fatalf("apply JSON command = stderr %q, exit %d; want stable progress on stderr and exit 0", stderr, code)
	}
}

func buildCLIProcess(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "cli-app")
	command := exec.Command("go", "build", "-o", binary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build CLI executable: %v\n%s", err, output)
	}
	return binary
}

func runCLIProcess(t *testing.T, binary string, args ...string) (string, string, int) {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	commandArgs := append([]string{"--config", configFile}, args...)
	command := exec.Command(binary, commandArgs...)
	command.Env = append(os.Environ(), "XDG_CONFIG_HOME="+filepath.Join(t.TempDir(), "no-user-config"))
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	code := 0
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("CLI process: %v; stderr = %q", err, stderr.String())
		}
		code = exitError.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}
