package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"charm.land/fang/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
)

// Execute runs root through Fang, which owns help rendering and error reporting.
// A signal cause recorded on ctx is joined to the result before it is printed once to
// the command's error writer, and the joined error is returned for exit-code mapping.
func Execute(ctx context.Context, root *cobra.Command) error {
	reported := false
	err := fang.Execute(ctx, root,
		fang.WithoutManpage(),
		fang.WithoutVersion(),
		fang.WithErrorHandler(func(w io.Writer, styles fang.Styles, err error) {
			reported = true
			reportError(w, styles, withCause(ctx, err))
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

// reportError prints the plain Stage 1 line unless the writer supports color (Fang hands the
// handler a colorprofile.Writer, so redirection and TERM=dumb land on the plain branch);
// otherwise it renders Fang's styled block plus a usage hint for validation errors.
// colorprofile.Env is consulted too because the writer's own detection lets CLICOLOR_FORCE
// override NO_COLOR on a non-TTY, while Env gives NO_COLOR precedence.
func reportError(w io.Writer, styles fang.Styles, err error) {
	if cw, ok := w.(*colorprofile.Writer); !ok || min(cw.Profile, colorprofile.Env(os.Environ())) <= colorprofile.ASCII {
		_, _ = fmt.Fprintln(w, err)
		return
	}
	fang.DefaultErrorHandler(w, styles, err)
	if errors.Is(err, app.ErrValidation) {
		text := styles.ErrorText.UnsetWidth().UnsetMargins().UnsetTransform()
		_, _ = fmt.Fprintln(w, text.Render("Try")+styles.Program.Flag.Render(" --help ")+text.Render("for usage."))
		_, _ = fmt.Fprintln(w)
	}
}
