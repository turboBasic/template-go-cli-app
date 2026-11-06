package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
)

// Execute runs root through Fang, which owns help rendering and error reporting.
// A signal cause recorded on ctx is joined to the result before it is printed once to
// the command's error writer, and the joined error is returned for exit-code mapping.
func Execute(ctx context.Context, root *cobra.Command) error {
	reported := false
	err := fang.Execute(ctx, root,
		fang.WithoutManpage(),
		fang.WithoutVersion(),
		fang.WithErrorHandler(func(w io.Writer, _ fang.Styles, err error) {
			reported = true
			_, _ = fmt.Fprintln(w, withCause(ctx, err))
		}),
	)
	joined := withCause(ctx, err)
	if joined != nil && !reported {
		_, _ = fmt.Fprintln(root.ErrOrStderr(), joined)
	}
	return joined
}

func withCause(ctx context.Context, err error) error {
	if cause := context.Cause(ctx); cause != nil {
		return errors.Join(err, cause)
	}
	return err
}
