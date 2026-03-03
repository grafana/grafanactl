package graph

import "github.com/charmbracelet/lipgloss"

// Grafana color palette for chart series.
//
//nolint:gochecknoglobals
var grafanaColors = []lipgloss.Color{
	lipgloss.Color("#7EB26D"), // Green
	lipgloss.Color("#EAB839"), // Yellow
	lipgloss.Color("#6ED0E0"), // Cyan
	lipgloss.Color("#EF843C"), // Orange
	lipgloss.Color("#E24D42"), // Red
	lipgloss.Color("#1F78C1"), // Blue
	lipgloss.Color("#BA43A9"), // Purple
	lipgloss.Color("#705DA0"), // Violet
	lipgloss.Color("#508642"), // Dark Green
	lipgloss.Color("#CCA300"), // Gold
}

// ColorForIndex returns the color for a given series index.
func ColorForIndex(idx int) lipgloss.Color {
	return grafanaColors[idx%len(grafanaColors)]
}
