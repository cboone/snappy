// Package tui implements the Bubbletea TUI for Snappy.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	bold    = lipgloss.NewStyle().Bold(true)
	dim     = lipgloss.NewStyle().Faint(true)
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	cyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	magenta = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	keyStyle   = lipgloss.NewStyle().Bold(true)
)

const (
	separator     = "================================================================================"
	thinSeparator = "------------------------------------------------------------------------------"
)
