package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Format selects the result encoding.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// ColorMode selects when text output may use terminal colors.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// TerminalCapabilities reports terminal support for each output stream.
type TerminalCapabilities struct {
	Stdout bool
	Stderr bool
}

// DetectTerminalCapabilities checks stdout and stderr independently.
func DetectTerminalCapabilities(stdout, stderr io.Writer) TerminalCapabilities {
	return detectTerminalCapabilities(stdout, stderr, isTerminal)
}

// ColorEnabled reports whether the selected text mode permits ANSI styling on a stream that is,
// or is not, a terminal.
func ColorEnabled(format Format, mode ColorMode, onTerminal, noColor bool) bool {
	if format == FormatJSON || mode == ColorNever {
		return false
	}
	if mode == ColorAlways {
		return true
	}
	return onTerminal && !noColor
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	return ok && isTerminalFile(file)
}

func detectTerminalCapabilities(stdout, stderr io.Writer, check func(io.Writer) bool) TerminalCapabilities {
	return TerminalCapabilities{Stdout: check(stdout), Stderr: check(stderr)}
}

// Options controls result formatting and terminal color policy.
type Options struct {
	Format    Format
	Color     ColorMode
	Terminals TerminalCapabilities
	NoColor   bool
}

// ProgressColorEnabled reports whether progress may use ANSI styling; progress goes to stderr, so
// the automatic mode follows stderr where results follow stdout. The result format does not apply: JSON
// constrains stdout, and progress never reaches it.
func (o Options) ProgressColorEnabled() bool {
	return ColorEnabled(FormatText, o.Color, o.Terminals.Stderr, o.NoColor)
}

// Renderer writes application results in the selected format.
type Renderer struct {
	Writer  io.Writer
	Options Options
}

// ColorEnabled reports whether this renderer may use ANSI styling.
func (r Renderer) ColorEnabled() bool {
	return ColorEnabled(r.Options.Format, r.Options.Color, r.Options.Terminals.Stdout, r.Options.NoColor)
}

// RenderResult writes value as JSON or delegates text formatting to renderText.
func (r Renderer) RenderResult(value any, renderText func(io.Writer, Options) error) error {
	if r.Options.Format == FormatJSON {
		if err := json.NewEncoder(r.Writer).Encode(value); err != nil {
			return fmt.Errorf("encode result: %w", err)
		}
		return nil
	}
	if renderText == nil {
		return fmt.Errorf("render text result: callback is nil")
	}
	if err := renderText(r.Writer, r.Options); err != nil {
		return fmt.Errorf("render text result: %w", err)
	}
	return nil
}
