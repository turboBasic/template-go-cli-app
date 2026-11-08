package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIProcessSeparatesJSONResultsAndDiagnostics(t *testing.T) {
	binary := buildCLIProcess(t, "")
	stdout, stderr, code := runCLIProcess(t, binary, "resource", "list")
	if code != 0 || stderr != "" || stdout != "NAME        STATUS\nexample-1   ready\nexample-2   pending\n" {
		t.Fatalf("resource list = stdout %q, stderr %q, exit %d; want deterministic fixture output and exit 0", stdout, stderr, code)
	}
	stdout, stderr, code = runCLIProcess(t, binary, "resource", "status", "invalid", "--output", "json")
	if stdout != "" || code != 1 || strings.Count(stderr, "resource not found") != 1 {
		t.Fatalf("invalid JSON command = stdout %q, stderr %q, exit %d; want empty stdout, diagnostic stderr, exit 1", stdout, stderr, code)
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

	stdout, stderr, code = runCLIProcess(t, binary, "resource", "apply", "example-1", "--output", "json", "--log-level", "debug")
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || result != (struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}{Name: "example-1", Status: "ready"}) {
		t.Fatalf("applied JSON = %q (decode error %v), want the existing resource result shape", stdout, err)
	}
	if code != 0 || !strings.Contains(stderr, "level=DEBUG") || !strings.Contains(stderr, "msg=\"applying resource\"") || !strings.Contains(stderr, "complete: ") {
		t.Fatalf("apply JSON command = stderr %q, exit %d; want debug logging and progress on stderr", stderr, code)
	}
}

func TestCLIProcessOnePasswordReference(t *testing.T) {
	const reference = "op://vault/item/field"
	binary := buildCLIProcess(t, "")
	stub := func(t *testing.T, body string) (pathDir, argvFile string) {
		t.Helper()
		pathDir = t.TempDir()
		argvFile = filepath.Join(pathDir, "argv")
		script := "#!/bin/sh\nprintf '%s ' \"$@\" > '" + argvFile + "'\n" + body + "\n"
		if err := os.WriteFile(filepath.Join(pathDir, "op"), []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
		return pathDir, argvFile
	}
	tests := []struct {
		name       string
		stub       string // body of the stub op; empty means no op on PATH
		credential string
		args       []string
		wantCode   int
		wantStderr string
		wantArgv   string // what the stub recorded; empty means it must not have run
	}{
		{name: "help needs no op", credential: reference, args: []string{"--help"}},
		{name: "completion needs no op", credential: reference, args: []string{"completion", "bash"}},
		{name: "apply without a reference needs no op", args: []string{"resource", "apply", "example-1"}},
		{name: "reference resolves through op", stub: "printf s3cr3t", credential: reference,
			args: []string{"resource", "apply", "example-1"}, wantArgv: "read --no-newline " + reference + " "},
		{name: "signed-out op fails with the reference", stub: "echo '[ERROR] You are not currently signed in.' >&2; exit 1", credential: reference,
			args: []string{"resource", "apply", "example-1"}, wantCode: 1, wantStderr: reference, wantArgv: "read --no-newline " + reference + " "},
		{name: "missing op is reported", credential: reference,
			args: []string{"resource", "apply", "example-1"}, wantCode: 1, wantStderr: "op is not installed"},
		{name: "reference without a scheme is rejected before op runs", stub: "printf s3cr3t", credential: "vault/item/field",
			args: []string{"resource", "apply", "example-1"}, wantCode: 2, wantStderr: "resource.credential"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathDir, argvFile := t.TempDir(), ""
			if tt.stub != "" {
				pathDir, argvFile = stub(t, tt.stub)
			}
			env := []string{"PATH=" + pathDir}
			if tt.credential != "" {
				env = append(env, "CLI_APP_RESOURCE_CREDENTIAL="+tt.credential)
			}
			stdout, stderr, code := runCLIProcessEnv(t, binary, env, tt.args...)
			if code != tt.wantCode || !strings.Contains(stderr, tt.wantStderr) {
				t.Fatalf("exit %d, stderr %q; want exit %d and stderr containing %q", code, stderr, tt.wantCode, tt.wantStderr)
			}
			if strings.Contains(stdout+stderr, "s3cr3t") {
				t.Fatalf("output contains the secret: stdout %q, stderr %q", stdout, stderr)
			}
			if argvFile == "" {
				return
			}
			argv, err := os.ReadFile(argvFile)
			if tt.wantArgv == "" {
				if err == nil {
					t.Fatalf("op ran with %q, want it never run", argv)
				}
				return
			}
			if err != nil || string(argv) != tt.wantArgv {
				t.Fatalf("op argv = %q (read error %v), want %q", argv, err, tt.wantArgv)
			}
		})
	}
}

func TestCLIProcessLongRunningDemoStopsOnSIGINT(t *testing.T) {
	binary := buildCLIProcess(t, "")
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "--config", configFile, "--color", "always", "demo", "long-running")
	command.Env = append(isolatedEnv(), "XDG_CONFIG_HOME="+filepath.Join(t.TempDir(), "no-user-config"))
	stderr, err := command.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill() })

	lines := bufio.NewScanner(stderr)
	if !lines.Scan() || !strings.Contains(lines.Text(), "working") {
		t.Fatalf("first stderr line = %q (scan error %v), want progress", lines.Text(), lines.Err())
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	output := lines.Text()
	finished := make(chan error, 1)
	go func() {
		for lines.Scan() {
			output += "\n" + lines.Text()
		}
		finished <- command.Wait()
	}()
	select {
	case err := <-finished:
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 130 {
			t.Fatalf("Wait() error = %v, want exit code 130", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("process kept running 10s after SIGINT, want it to stop between steps")
	}
	if got := strings.Count(output, "interrupted by SIGINT"); got != 1 || strings.Contains(output, "✓") {
		t.Fatalf("stderr = %q, want one interrupt report and no completion line", output)
	}
}

func TestCLIProcessLongRunningDemoTimeoutExits124(t *testing.T) {
	binary := buildCLIProcess(t, "")
	started := time.Now()
	stdout, stderr, code := runCLIProcess(t, binary, "demo", "long-running", "--timeout", "100ms")
	if code != 124 || stdout != "" || time.Since(started) > 10*time.Second {
		t.Fatalf("demo long-running --timeout 100ms = stdout %q, stderr %q, exit %d after %s; want exit 124 well before the demo's length", stdout, stderr, code, time.Since(started))
	}
}

func buildCLIProcess(t *testing.T, ldflags string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "cli-app")
	command := exec.Command("go", "build", "-ldflags", ldflags, "-o", binary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build CLI executable: %v\n%s", err, output)
	}
	return binary
}

func runCLIProcess(t *testing.T, binary string, args ...string) (string, string, int) {
	t.Helper()
	return runCLIProcessEnv(t, binary, nil, args...)
}

// runCLIProcessEnv runs the CLI with extraEnv added after the isolated environment, so it wins.
func runCLIProcessEnv(t *testing.T, binary string, extraEnv []string, args ...string) (string, string, int) {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "settings.toml")
	if err := os.WriteFile(configFile, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	commandArgs := append([]string{"--config", configFile}, args...)
	command := exec.Command(binary, commandArgs...)
	command.Env = append(append(isolatedEnv(), "XDG_CONFIG_HOME="+filepath.Join(t.TempDir(), "no-user-config")), extraEnv...)
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

// isolatedEnv drops the settings that change results or color, so results do not depend on the caller's shell.
func isolatedEnv() []string {
	var env []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "CLI_APP_") || strings.HasPrefix(name, "CLICOLOR") || name == "NO_COLOR" || name == "TERM" {
			continue
		}
		env = append(env, entry)
	}
	return env
}
