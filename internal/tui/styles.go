// Package tui implements the Bubbletea TUI for Snappy.
package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

const (
	indicatorOn      = "●"
	indicatorOff     = "○"
	indicatorPurge   = "◇"
	indicatorPinned  = "◆"
	indicatorWarning = "⚠"
)

func adaptiveColor(hasDarkBG bool, light, dark string) color.Color {
	return lipgloss.LightDark(hasDarkBG)(lipgloss.Color(light), lipgloss.Color(dark))
}

// modelStyles holds all precomputed Lipgloss styles for the TUI.
type modelStyles struct {
	textDim     lipgloss.Style
	textGreen   lipgloss.Style
	textYellow  lipgloss.Style
	textRed     lipgloss.Style
	textCyan    lipgloss.Style
	textMagenta lipgloss.Style

	section      lipgloss.Style
	sectionFocus lipgloss.Style
	sectionTitle lipgloss.Style
	helpBar      lipgloss.Style
	statusOn     lipgloss.Style
	statusOff    lipgloss.Style
	snapNumber   lipgloss.Style
	spinnerStyle lipgloss.Style
}

func newModelStyles(hasDarkBG bool) modelStyles {
	colorBorder := adaptiveColor(hasDarkBG, "#888888", "#555555")
	colorHighlight := adaptiveColor(hasDarkBG, "#0066cc", "#88c0d0")
	colorSubtle := adaptiveColor(hasDarkBG, "#666666", "#777777")
	green := lipgloss.Color("2")
	yellow := lipgloss.Color("3")
	red := lipgloss.Color("1")
	cyan := lipgloss.Color("6")
	magenta := lipgloss.Color("5")

	return modelStyles{
		textDim:     lipgloss.NewStyle().Faint(true),
		textGreen:   lipgloss.NewStyle().Foreground(green),
		textYellow:  lipgloss.NewStyle().Foreground(yellow),
		textRed:     lipgloss.NewStyle().Foreground(red),
		textCyan:    lipgloss.NewStyle().Foreground(cyan),
		textMagenta: lipgloss.NewStyle().Foreground(magenta),

		section: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1),

		sectionFocus: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorHighlight).
			Padding(0, 1),

		sectionTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHighlight),

		helpBar: lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 1),

		statusOn:  lipgloss.NewStyle().Bold(true).Foreground(green),
		statusOff: lipgloss.NewStyle().Bold(true).Foreground(red),

		snapNumber:   lipgloss.NewStyle().Bold(true).Foreground(green),
		spinnerStyle: lipgloss.NewStyle().Foreground(colorHighlight),
	}
}

// contentWidth returns the usable width inside a bordered section.
// Accounts for border (2) + padding (2) = 4 characters.
func contentWidth(termWidth int) int {
	w := termWidth - 4
	if w < 40 {
		return 40
	}
	return w
}

// flexPanelHeights calculates the viewport heights for the snapshot and log
// panels by subtracting fixed-height sections from the terminal height, then
// splitting the remaining space equally (1:1).
func flexPanelHeights(termHeight, fixedHeight int) (snapH, logH int) {
	flexible := termHeight - fixedHeight
	if flexible < 6 {
		flexible = 6
	}
	snapH = flexible / 2
	logH = flexible - snapH
	return snapH, logH
}
