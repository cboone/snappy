// Package tui implements the Bubbletea TUI for Snappy.
package tui

import (
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

const (
	indicatorOn      = "●"
	indicatorOff     = "○"
	indicatorWarning = "⚠"
)

// modelStyles holds all precomputed Lipgloss styles for the TUI.
type modelStyles struct {
	textDim     lipgloss.Style
	textGreen   lipgloss.Style
	textYellow  lipgloss.Style
	textRed     lipgloss.Style
	textCyan    lipgloss.Style
	textMagenta lipgloss.Style

	infoLabel lipgloss.Style

	section      lipgloss.Style
	sectionFocus lipgloss.Style
	sectionTitle lipgloss.Style
	helpBar      lipgloss.Style
	statusOn     lipgloss.Style
	statusOff    lipgloss.Style
	spinnerStyle lipgloss.Style
	tableStyles  table.Styles
}

func newModelStyles(hasDarkBG bool) modelStyles {
	lightDark := lipgloss.LightDark(hasDarkBG)

	colorBorder := lightDark(lipgloss.Color("245"), lipgloss.Color("240"))
	colorHighlight := lightDark(lipgloss.Color("26"), lipgloss.Color("110"))
	colorSubtle := lightDark(lipgloss.Color("241"), lipgloss.Color("243"))
	green := lipgloss.Green
	yellow := lipgloss.Yellow
	red := lipgloss.Red
	cyan := lipgloss.Cyan
	magenta := lipgloss.Magenta

	return modelStyles{
		textDim:     lipgloss.NewStyle().Faint(true),
		infoLabel:   lipgloss.NewStyle().Foreground(lightDark(lipgloss.Color("240"), lipgloss.Color("248"))),
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

		spinnerStyle: lipgloss.NewStyle().Foreground(colorHighlight),
		tableStyles: table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Foreground(colorSubtle).Padding(0, 1),
			Cell:     lipgloss.NewStyle().Padding(0, 1),
			Selected: lipgloss.NewStyle().Bold(true).Foreground(colorHighlight),
		},
	}
}

// contentWidth returns the usable text width inside a bordered, padded
// section. The floor (56) guarantees all five table columns fit at
// minimum widths (DATE 19 + AGE 9 + XID 8 + UUID 9 + STATUS 1 + 10 pad).
// Render functions derive section Width as cw + 4 so lipgloss wraps at
// exactly this content width.
func contentWidth(termWidth int) int {
	w := termWidth - 4
	if w < 56 {
		return 56
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
