package onepassword

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// killWait bounds how long Run waits for output pipes after the context kills the process.
const killWait = time.Second

// ExecRunner runs commands as child processes that inherit the environment, which is how op
// finds its configured account. The zero value is ready to use.
type ExecRunner struct{}

var _ Runner = ExecRunner{}

// Run executes name with args, without a shell, and kills the process when ctx ends.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error) {
	var out, errOut bytes.Buffer
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout, command.Stderr = &out, &errOut
	command.WaitDelay = killWait
	err = command.Run()
	return out.Bytes(), errOut.Bytes(), err
}
