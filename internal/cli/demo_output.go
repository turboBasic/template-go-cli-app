package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/turboBasic/template-go-cli-app/internal/domain"
	"github.com/turboBasic/template-go-cli-app/internal/output"
)

type demoResourceResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func renderDemoResource(w io.Writer, options output.Options, resource domain.Resource) error {
	status := resource.Status
	if output.ColorEnabled(options.Format, options.Color, options.Terminals.Stdout, options.NoColor) {
		status = output.StyleStatus(status)
	}
	_, err := fmt.Fprintf(w, "%s: %s\n", resource.Name, status)
	return err
}

func renderDemoResources(w io.Writer, options output.Options, resources []domain.Resource) error {
	var table bytes.Buffer
	writer := tabwriter.NewWriter(&table, 12, 8, 1, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tSTATUS"); err != nil {
		return err
	}
	for _, resource := range resources {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", resource.Name, resource.Status); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	text := table.String()
	if output.ColorEnabled(options.Format, options.Color, options.Terminals.Stdout, options.NoColor) {
		lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
		statusOffset := strings.Index(lines[0], "STATUS")
		lines[0] = output.StyleHeader(lines[0])
		for i := 1; i < len(lines); i++ {
			if statusOffset >= 0 && len(lines[i]) >= statusOffset {
				lines[i] = lines[i][:statusOffset] + output.StyleStatus(lines[i][statusOffset:])
			}
		}
		text = strings.Join(lines, "\n") + "\n"
	}
	_, err := io.WriteString(w, text)
	return err
}

func demoResourceResultFor(resource domain.Resource) demoResourceResult {
	return demoResourceResult{Name: resource.Name, Status: resource.Status}
}

func demoResourceResultsFor(resources []domain.Resource) []demoResourceResult {
	results := make([]demoResourceResult, len(resources))
	for i, resource := range resources {
		results[i] = demoResourceResultFor(resource)
	}
	return results
}
