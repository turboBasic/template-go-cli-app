package output

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

const barCells = 50

func renderBar(fraction float64) string {
	fraction = math.Max(0, math.Min(1, fraction))
	filled := int(math.Round(fraction * barCells))
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(strings.Repeat("░", barCells-filled))
	return fmt.Sprintf("%s %3.0f%%", bar, fraction*100)
}
