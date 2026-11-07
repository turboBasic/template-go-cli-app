package output

import "charm.land/lipgloss/v2"

// styles holds the basic-ANSI styles shared by result rendering. Basic colors keep
// `--color always` bytes independent of the terminal.
type styles struct {
	header  lipgloss.Style
	ready   lipgloss.Style
	pending lipgloss.Style
	phase   lipgloss.Style
	detail  lipgloss.Style
}

func newStyles() styles {
	return styles{
		header:  lipgloss.NewStyle().Bold(true),
		ready:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		pending: lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		phase:   lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		detail:  lipgloss.NewStyle().Faint(true),
	}
}

// status styles a known status and returns any other value unchanged.
func (s styles) status(value string) string {
	switch value {
	case "ready":
		return s.ready.Render(value)
	case "pending":
		return s.pending.Render(value)
	}
	return value
}
