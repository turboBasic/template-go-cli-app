package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/turboBasic/template-go-cli-app/internal/config"
)

// RenderConfig writes effective configuration entries with the source of each value.
func (r Renderer) RenderConfig(entries []config.Entry) error {
	if r.Options.Format == FormatJSON {
		items := make([]struct {
			Key    string `json:"key"`
			Value  any    `json:"value"`
			Source string `json:"source"`
			Origin string `json:"origin,omitempty"`
		}, len(entries))
		for i, entry := range entries {
			items[i].Key, items[i].Value, items[i].Source, items[i].Origin = entry.Key, entry.Value, string(entry.Source), entry.Origin
		}
		if err := json.NewEncoder(r.Writer).Encode(items); err != nil {
			return fmt.Errorf("encode configuration: %w", err)
		}
		return nil
	}

	var table bytes.Buffer
	writer := tabwriter.NewWriter(&table, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "KEY\tVALUE\tSOURCE"); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	for _, entry := range entries {
		value := fmt.Sprintf("%v", entry.Value)
		if text, ok := entry.Value.(string); ok {
			value = fmt.Sprintf("%q", text)
		}
		source := string(entry.Source)
		if entry.Origin != "" {
			source += " (" + entry.Origin + ")"
		}
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\n", entry.Key, value, source); err != nil {
			return fmt.Errorf("write configuration: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush configuration: %w", err)
	}
	out := table.String()
	if r.ColorEnabled() {
		line, rest, _ := strings.Cut(out, "\n")
		out = newStyles().header.Render(line) + "\n" + rest
	}
	if _, err := io.WriteString(r.Writer, out); err != nil {
		return fmt.Errorf("flush configuration: %w", err)
	}
	return nil
}
