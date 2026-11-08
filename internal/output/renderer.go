package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
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

// Renderer writes resource results in the selected format.
type Renderer struct {
	Writer  io.Writer
	Options Options
}

// ColorEnabled reports whether this renderer may use ANSI styling.
func (r Renderer) ColorEnabled() bool {
	return ColorEnabled(r.Options.Format, r.Options.Color, r.Options.Terminals.Stdout, r.Options.NoColor)
}

// RenderList writes a resource list.
func (r Renderer) RenderList(resources []domain.Resource) error {
	if r.Options.Format == FormatJSON {
		items := make([]struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}, len(resources))
		for i, resource := range resources {
			items[i].Name = resource.Name
			items[i].Status = resource.Status
		}
		if err := json.NewEncoder(r.Writer).Encode(items); err != nil {
			return fmt.Errorf("encode resource list: %w", err)
		}
		return nil
	}

	var table bytes.Buffer
	writer := tabwriter.NewWriter(&table, 12, 8, 1, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tSTATUS"); err != nil {
		return fmt.Errorf("write resource list: %w", err)
	}
	for _, resource := range resources {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", resource.Name, resource.Status); err != nil {
			return fmt.Errorf("write resource list: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush resource list: %w", err)
	}
	out := table.String()
	if r.ColorEnabled() {
		out = styleTable(out)
	}
	if _, err := io.WriteString(r.Writer, out); err != nil {
		return fmt.Errorf("flush resource list: %w", err)
	}
	return nil
}

// styleTable styles an aligned table after the fact, so escapes never reach the tabwriter
// and column alignment is the plain one. The STATUS column offset comes from the header line.
func styleTable(table string) string {
	s := newStyles()
	lines := strings.Split(strings.TrimSuffix(table, "\n"), "\n")
	offset := strings.Index(lines[0], "STATUS")
	lines[0] = s.header.Render(lines[0])
	for i := 1; i < len(lines); i++ {
		if offset >= 0 && len(lines[i]) >= offset {
			lines[i] = lines[i][:offset] + s.status(lines[i][offset:])
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// RenderResource writes one resource.
func (r Renderer) RenderResource(resource domain.Resource) error {
	if r.Options.Format == FormatJSON {
		if err := json.NewEncoder(r.Writer).Encode(struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}{Name: resource.Name, Status: resource.Status}); err != nil {
			return fmt.Errorf("encode resource: %w", err)
		}
		return nil
	}
	status := resource.Status
	if r.ColorEnabled() {
		status = newStyles().status(status)
	}
	if _, err := fmt.Fprintf(r.Writer, "%s: %s\n", resource.Name, status); err != nil {
		return fmt.Errorf("write resource: %w", err)
	}
	return nil
}
