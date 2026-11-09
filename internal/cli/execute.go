package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"charm.land/fang/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"github.com/turboBasic/template-go-cli-app/internal/app"
)

// Execute runs root through Fang, which owns help rendering and error reporting.
// A signal cause recorded on ctx is joined to the result before it is printed once to
// the command's error writer, and the joined error is returned for exit-code mapping.
func Execute(ctx context.Context, root *cobra.Command) error {
	// Cobra binds the completion scripts' output writer when it creates the command, so it is
	// created here, after the caller has set the writers, rather than when the tree is built.
	root.InitDefaultCompletionCmd()
	for _, command := range root.Commands() {
		if command.Name() == "completion" {
			rejectUnknownSubcommands(command)
			for _, shell := range command.Commands() {
				shell.Args = validationArgs(shell.Args)
			}
		}
	}
	postRun := root.PersistentPostRunE
	root.PersistentPostRunE = nil
	defer func() { root.PersistentPostRunE = postRun }()
	finished := false
	finish := func() error {
		if finished {
			return nil
		}
		finished = true
		if postRun == nil {
			return nil
		}
		return postRun(root, nil)
	}
	reported := false
	var reportedErr error
	err := fang.Execute(ctx, root,
		fang.WithoutManpage(),
		fang.WithoutVersion(),
		fang.WithErrorHandler(func(w io.Writer, styles fang.Styles, err error) {
			reported = true
			reportedErr = errors.Join(err, finish())
			reportError(w, styles, withCause(ctx, reportedErr))
		}),
	)
	if reported {
		err = reportedErr
	} else {
		err = errors.Join(err, finish())
	}
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
func reportError(w io.Writer, styles fang.Styles, err error) {
	if cw, ok := w.(*colorprofile.Writer); !ok || cw.Profile <= colorprofile.ASCII {
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
