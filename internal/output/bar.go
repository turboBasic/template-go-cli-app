package output

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
)

const (
	barCells = 50
	barFPS   = 60
	// partialBlocks are the eighths of a cell, from one eighth to seven.
	partialBlocks = "▏▎▍▌▋▊▉"
)

// frameMsg advances the bar's glide by one frame.
type frameMsg struct{}

// barModel draws a bar that moves in eighths of a cell and glides to each new target on a spring,
// so the fill keeps moving between steps instead of jumping a cell at a time. The fill blends
// purple to pink across the bar, so the color shows how far the work has come.
type barModel struct {
	fills                   []color.Color
	spring                  harmonica.Spring
	shown, velocity, target float64
	ticking                 bool
}

func newBar() barModel {
	return barModel{
		fills: lipgloss.Blend1D(barCells, lipgloss.Color("#5A56E0"), lipgloss.Color("#EE6FF8")),
		// A soft spring (frequency 8) keeps the fill moving until the next step arrives.
		spring: harmonica.NewSpring(harmonica.FPS(barFPS), 8, 1),
	}
}

// viewAs draws the bar at fraction (0 to 1) with its percentage, without any animation.
func (b barModel) viewAs(fraction float64) string {
	fraction = math.Max(0, math.Min(1, fraction))
	eighths := int(math.Round(fraction * barCells * 8))
	full, part := eighths/8, eighths%8
	var out strings.Builder
	cell := func(i int, glyph string) {
		out.WriteString(lipgloss.NewStyle().Foreground(b.fills[i]).Render(glyph))
	}
	for i := range full {
		cell(i, "█")
	}
	empty := barCells - full
	if part > 0 {
		cell(full, string([]rune(partialBlocks)[part-1]))
		empty--
	}
	out.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("░", empty)))
	fmt.Fprintf(&out, " %3.0f%%", fraction*100)
	return out.String()
}

// view draws the bar where the glide has reached.
func (b barModel) view() string { return b.viewAs(b.shown) }

// setTarget moves the glide's target and starts its frames unless they are already running.
func (b *barModel) setTarget(fraction float64) tea.Cmd {
	b.target = fraction
	if b.ticking {
		return nil
	}
	b.ticking = true
	return frameTick()
}

// frame advances the glide one step, and stops the frames once it has settled on the target.
func (b *barModel) frame() tea.Cmd {
	b.shown, b.velocity = b.spring.Update(b.shown, b.velocity, b.target)
	// 0.0005 is a fifth of an eighth of a cell, so the final snap onto the target cannot be seen.
	if math.Abs(b.shown-b.target) < 0.0005 && math.Abs(b.velocity) < 0.01 {
		b.shown, b.velocity, b.ticking = b.target, 0, false
		return nil
	}
	return frameTick()
}

func frameTick() tea.Cmd {
	return tea.Tick(time.Second/barFPS, func(time.Time) tea.Msg { return frameMsg{} })
}
