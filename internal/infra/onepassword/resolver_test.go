package onepassword

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/turboBasic/template-go-cli-app/internal/app"
)

const (
	testReference = "op://vault/item/field"
	testSecret    = "s3cr3t"
)

type fakeRunner struct {
	stdout, stderr string
	err            error
	block          bool
	calls          [][]string
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if f.block {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	return []byte(f.stdout), []byte(f.stderr), f.err
}

func TestResolve(t *testing.T) {
	t.Parallel()
	exitError := errors.New("exit status 1")
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	tests := []struct {
		name      string
		ctx       context.Context
		reference string
		runner    *fakeRunner
		want      string
		wantIs    error   // nil means a plain error or success
		wantNot   []error // categories the error must not carry
		wantErr   bool
		wantCalls int
		wantArgv  []string
	}{
		{name: "success", reference: testReference, runner: &fakeRunner{stdout: testSecret}, want: testSecret, wantCalls: 1,
			wantArgv: []string{"op", "read", "--no-newline", testReference}},
		{name: "missing item", reference: testReference, wantErr: true, wantCalls: 1,
			runner: &fakeRunner{stdout: testSecret, err: exitError, stderr: `[ERROR] "item" isn't an item in the "vault" vault.`},
			wantIs: app.ErrValidation, wantNot: []error{app.ErrAuthentication, app.ErrUnavailable}},
		{name: "missing vault", reference: testReference, wantErr: true, wantCalls: 1,
			runner: &fakeRunner{stdout: testSecret, err: exitError, stderr: `[ERROR] "vault" isn't a vault in this account.`},
			wantIs: app.ErrValidation, wantNot: []error{app.ErrAuthentication, app.ErrUnavailable}},
		{name: "missing field", reference: testReference, wantErr: true, wantCalls: 1,
			runner: &fakeRunner{stdout: testSecret, err: exitError, stderr: `[ERROR] could not read secret: "field" isn't a field in the "item" item`},
			wantIs: app.ErrValidation, wantNot: []error{app.ErrAuthentication, app.ErrUnavailable}},
		{name: "authentication failure", reference: testReference, wantErr: true, wantCalls: 1,
			runner: &fakeRunner{stdout: testSecret, err: exitError, stderr: "[ERROR] You are not currently signed in. Run 'op signin'."},
			wantIs: app.ErrAuthentication, wantNot: []error{app.ErrValidation, app.ErrUnavailable}},
		{name: "binary not found", reference: testReference, wantErr: true, wantCalls: 1,
			runner:  &fakeRunner{stdout: testSecret, err: fmt.Errorf("start: %w", exec.ErrNotFound)},
			wantNot: []error{app.ErrValidation, app.ErrAuthentication, app.ErrUnavailable}},
		{name: "unrecognised stderr", reference: testReference, wantErr: true, wantCalls: 1,
			runner:  &fakeRunner{stdout: testSecret, err: exitError, stderr: "[ERROR] network is unreachable"},
			wantNot: []error{app.ErrValidation, app.ErrAuthentication, app.ErrUnavailable}},
		{name: "reference without op scheme", reference: "https://vault/item/field", runner: &fakeRunner{stdout: testSecret}, wantErr: true,
			wantIs: app.ErrValidation, wantNot: []error{app.ErrAuthentication, app.ErrUnavailable}},
		{name: "cancelled before the call", ctx: cancelled, reference: testReference, wantErr: true, wantCalls: 1,
			runner: &fakeRunner{stdout: testSecret, block: true}, wantIs: context.Canceled,
			wantNot: []error{app.ErrValidation, app.ErrAuthentication, app.ErrUnavailable}},
		{name: "cancelled while op reports an authentication prompt", ctx: cancelled, reference: testReference, wantErr: true, wantCalls: 1,
			runner: &fakeRunner{stdout: testSecret, err: exitError, stderr: "authorization prompt dismissed"}, wantIs: context.Canceled,
			wantNot: []error{app.ErrValidation, app.ErrAuthentication, app.ErrUnavailable}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			got, err := NewResolver(tt.runner).Resolve(ctx, tt.reference)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Resolve error = %v, wantErr %t", err, tt.wantErr)
			}
			if err == nil && string(got) != tt.want {
				t.Fatalf("Resolve = %q, want %q", string(got), tt.want)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("Resolve error = %v, want it to match %v", err, tt.wantIs)
			}
			for _, other := range tt.wantNot {
				if errors.Is(err, other) {
					t.Fatalf("Resolve error = %v, must not match %v", err, other)
				}
			}
			if err != nil {
				if strings.Contains(err.Error(), testSecret) {
					t.Fatalf("error text contains the secret: %q", err)
				}
				if tt.wantIs != context.Canceled && !strings.Contains(err.Error(), tt.reference) {
					t.Fatalf("error text %q does not name the reference", err)
				}
			}
			if len(tt.runner.calls) != tt.wantCalls {
				t.Fatalf("runner calls = %d, want %d", len(tt.runner.calls), tt.wantCalls)
			}
			if tt.wantArgv != nil && !reflect.DeepEqual(tt.runner.calls[0], tt.wantArgv) {
				t.Fatalf("argv = %v, want %v", tt.runner.calls[0], tt.wantArgv)
			}
		})
	}
}
