package onepassword

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

const helperModeEnv = "ONEPASSWORD_HELPER_MODE"

// TestExecRunnerHelper is the child process the ExecRunner tests start; it does nothing in a
// normal test run.
func TestExecRunnerHelper(t *testing.T) {
	switch os.Getenv(helperModeEnv) {
	case "":
		return
	case "ok":
		_, _ = fmt.Fprint(os.Stdout, "out")
		fmt.Fprint(os.Stderr, "err")
		os.Exit(0)
	case "fail":
		_, _ = fmt.Fprint(os.Stdout, "out")
		fmt.Fprint(os.Stderr, "boom")
		os.Exit(3)
	case "block":
		time.Sleep(time.Minute)
	}
}

func runHelper(t *testing.T, ctx context.Context, mode string) (stdout, stderr []byte, err error) {
	t.Helper()
	t.Setenv(helperModeEnv, mode)
	return ExecRunner{}.Run(ctx, os.Args[0], "-test.run=^TestExecRunnerHelper$")
}

func TestExecRunner(t *testing.T) {
	t.Run("captures stdout and stderr separately", func(t *testing.T) {
		stdout, stderr, err := runHelper(t, context.Background(), "ok")
		if err != nil {
			t.Fatalf("Run error = %v", err)
		}
		if string(stdout) != "out" || string(stderr) != "err" {
			t.Fatalf("stdout = %q, stderr = %q, want %q and %q", stdout, stderr, "out", "err")
		}
	})

	t.Run("returns the exit error with stderr intact", func(t *testing.T) {
		_, stderr, err := runHelper(t, context.Background(), "fail")
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 3 {
			t.Fatalf("Run error = %v, want an *exec.ExitError with code 3", err)
		}
		if string(stderr) != "boom" {
			t.Fatalf("stderr = %q, want %q", stderr, "boom")
		}
	})

	t.Run("reports a missing binary as exec.ErrNotFound", func(t *testing.T) {
		_, _, err := ExecRunner{}.Run(context.Background(), "onepassword-test-no-such-binary")
		if !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf("Run error = %v, want exec.ErrNotFound", err)
		}
	})

	t.Run("kills the child when the context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			_, _, err := runHelper(t, ctx, "block")
			done <- err
		}()
		time.Sleep(200 * time.Millisecond)
		cancel()
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("Run returned nil after cancellation")
			}
		case <-time.After(time.Second):
			t.Fatal("Run did not return within a second of cancellation")
		}
	})
}
