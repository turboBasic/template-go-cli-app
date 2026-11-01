package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/cli"
	"github.com/turboBasic/template-go-cli-app/internal/infra/memory"
)

func TestCLIProcessHelper(t *testing.T) {
	if os.Getenv("CLI_APP_CLI_HELPER") != "1" {
		return
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "no-user-config"))
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	resources := memory.NewResourceReader()
	command := cli.NewRootCommand(resources, cli.BuildInfo{}, resources.Names())
	args := []string{"--config", configFile, "resource"}
	if os.Getenv("CLI_APP_CLI_HELPER_MODE") == "invalid" {
		args = append(args, "apply", "invalid", "--output", "json")
	} else {
		args = append(args, "status", "example-1", "--output", "json")
	}
	command.SetArgs(args)
	command.SetOut(os.Stdout)
	command.SetErr(os.Stderr)
	code := execute(func() error { return runCommand(command) }, os.Stderr)
	if err := os.WriteFile(os.Getenv("CLI_APP_CLI_HELPER_REPORT"), []byte(strconv.Itoa(code)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCLIProcessSeparatesJSONResultsAndDiagnostics(t *testing.T) {
	stdout, stderr, code := runCLIProcess(t, "invalid")
	if stdout != "" || code != 2 || !strings.Contains(stderr, "unknown resource") {
		t.Fatalf("invalid JSON command = stdout %q, stderr %q, exit %d; want empty stdout, diagnostic stderr, exit 2", stdout, stderr, code)
	}

	stdout, stderr, code = runCLIProcess(t, "success")
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
}

func runCLIProcess(t *testing.T, mode string) (string, string, int) {
	t.Helper()
	report := filepath.Join(t.TempDir(), "exit-code")
	command := exec.Command(os.Args[0], "-test.run=^TestCLIProcessHelper$")
	command.Env = append(os.Environ(),
		"CLI_APP_CLI_HELPER=1",
		"CLI_APP_CLI_HELPER_MODE="+mode,
		"CLI_APP_CLI_HELPER_REPORT="+report,
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("CLI helper process: %v; stderr = %q", err, stderr.String())
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read CLI helper exit code: %v", err)
	}
	code, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("parse CLI helper exit code %q: %v", data, err)
	}
	// The test binary appends PASS after the helper's application output.
	return strings.TrimSuffix(stdout.String(), "PASS\n"), stderr.String(), code
}
