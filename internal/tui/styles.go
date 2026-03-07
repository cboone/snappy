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
	indicatorPinned  = "📌"
)

// modelStyles holds all precomputed Lipgloss styles for the TUI.
type modelStyles struct {
	textDefault lipgloss.Style
	textDim     lipgloss.Style
	textGreen   lipgloss.Style
	textYellow  lipgloss.Style
	textRed     lipgloss.Style
	textCyan    lipgloss.Style
	textMagenta lipgloss.Style

	infoLabel lipgloss.Style

	section         lipgloss.Style
	sectionFocus    lipgloss.Style
	sectionTitle    lipgloss.Style
	sectionTitleDim lipgloss.Style
	helpBar         lipgloss.Style
	statusOn        lipgloss.Style
	statusOff       lipgloss.Style
	spinnerStyle    lipgloss.Style
	tableStyles     table.Styles
}

func newModelStyles(hasDarkBG bool) modelStyles {
	lightDark := lipgloss.LightDark(hasDarkBG)

	colorBorder := lightDark(lipgloss.Color("250"), lipgloss.Color("235"))
	colorFocus := lightDark(lipgloss.Black, lipgloss.White)
	colorHighlight := lightDark(lipgloss.Color("26"), lipgloss.Color("110"))
	colorTitleDim := lightDark(lipgloss.Color("247"), lipgloss.Color("240"))
	colorSubtle := lightDark(lipgloss.Color("243"), lipgloss.Color("241"))
	colorLabel := lightDark(lipgloss.Color("240"), lipgloss.Color("248"))
	colorHeading := lightDark(lipgloss.Color("245"), lipgloss.Color("243"))
	green := lipgloss.Green
	yellow := lipgloss.Yellow
	red := lipgloss.Red
	cyan := lipgloss.Cyan
	magenta := lipgloss.Magenta

	return modelStyles{
		textDefault: lipgloss.NewStyle(),
		textDim:     lipgloss.NewStyle().Faint(true),
		infoLabel:   lipgloss.NewStyle().Foreground(colorLabel),
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
			BorderForeground(colorFocus).
			Padding(0, 1),

		sectionTitle: lipgloss.NewStyle().
			Bold(true),

		sectionTitleDim: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTitleDim),

		helpBar: lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 1),

		statusOn:  lipgloss.NewStyle().Bold(true).Foreground(green),
		statusOff: lipgloss.NewStyle().Bold(true).Foreground(red),

		spinnerStyle: lipgloss.NewStyle().Foreground(colorHighlight),
		tableStyles: table.Styles{
			Header:   lipgloss.NewStyle().Bold(true).Foreground(colorHeading).Padding(0, 3, 0, 0),
			Cell:     lipgloss.NewStyle().Padding(0, 3, 0, 0),
			Selected: lipgloss.NewStyle().Bold(true).Foreground(colorHighlight).Padding(0, 3, 0, 0),
		},
	}
}

// contentWidth returns the usable text width inside a bordered, padded
// section. The floor (65) guarantees all six table columns fit at
// minimum widths (DATE 19 + AGE 5 + XID 7 + DELTA 7 + UUID 9 + STATUS 0
// + 18 pad). Padding is 3 per column (right only), so 6 columns = 18.
// Render functions derive section Width as cw + 4 so lipgloss wraps at
// exactly this content width.
func contentWidth(termWidth int) int {
	w := termWidth - 4
	if w < 65 {
		return 65
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
