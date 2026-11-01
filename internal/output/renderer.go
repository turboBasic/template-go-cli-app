package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	return TerminalCapabilities{Stdout: isTerminal(stdout), Stderr: isTerminal(stderr)}
}

// ColorEnabled reports whether the selected text mode permits ANSI styling.
func ColorEnabled(format Format, mode ColorMode, terminals TerminalCapabilities, noColor bool) bool {
	if format == FormatJSON || mode == ColorNever {
		return false
	}
	if mode == ColorAlways {
		return true
	}
	return terminals.Stdout && !noColor
}

type fileWithStat interface {
	Stat() (os.FileInfo, error)
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(fileWithStat)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// Renderer writes resource results in the selected format.
type Renderer struct {
	Writer io.Writer
	Format Format
}

// RenderList writes a resource list.
func (r Renderer) RenderList(resources []domain.Resource) error {
	if r.Format == FormatJSON {
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

	writer := tabwriter.NewWriter(r.Writer, 12, 8, 0, ' ', 0)
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
	return nil
}

// RenderResource writes one resource.
func (r Renderer) RenderResource(resource domain.Resource) error {
	if r.Format == FormatJSON {
		if err := json.NewEncoder(r.Writer).Encode(struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}{Name: resource.Name, Status: resource.Status}); err != nil {
			return fmt.Errorf("encode resource: %w", err)
		}
		return nil
	}
	if _, err := fmt.Fprintf(r.Writer, "%s: %s\n", resource.Name, resource.Status); err != nil {
		return fmt.Errorf("write resource: %w", err)
	}
	return nil
}
