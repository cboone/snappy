package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// View renders the full TUI screen as a tea.View.
func (m Model) View() tea.View {
	w := m.width
	if w == 0 {
		w = 80
	}

	infoPanel := m.renderInfoPanel(w)
	snapPanel := m.renderSnapshotPanel(w)
	logPanel := m.renderLogPanel(w)
	helpBar := m.renderHelpBar(w)

	content := lipgloss.JoinVertical(lipgloss.Left,
		infoPanel,
		snapPanel,
		logPanel,
		helpBar,
	)

	v := tea.NewView(content)
	if !m.quitting {
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

func (m Model) buildDotIndicator() string {
	dot := indicatorOff
	if m.auto.Enabled() {
		dot = indicatorOn
	}
	if m.loading {
		return m.styles.textYellow.Render(dot)
	}
	return m.styles.textGreen.Render(dot)
}

func (m Model) buildSpinnerSuffix() string {
	switch {
	case m.snapshotting:
		return "  Snapshotting " + m.spinner.View()
	case m.thinning:
		return "  Thinning " + m.spinner.View()
	case m.loading:
		return "  Refreshing " + m.spinner.View()
	default:
		return ""
	}
}

func (m Model) renderInfoPanel(width int) string {
	cw := contentWidth(width)

	dot := m.buildDotIndicator()
	spinnerSuffix := m.buildSpinnerSuffix()

	// Build the info panel body.
	lastRefresh := "never"
	if !m.lastRefresh.IsZero() {
		lastRefresh = m.lastRefresh.Format("2006-01-02T15:04:05")
	}

	diskInfo := m.diskInfo
	if diskInfo == "" {
		diskInfo = "unavailable"
	}

	label := m.styles.infoLabel.Render
	line1 := label("Volume:") + " " + m.volumeName + "    " + label("Disk:") + " " + diskInfo
	if m.tidemark != "" {
		line1 += "    " + label("Tidemark:") + " " + m.tidemark
	}
	lines := []string{
		line1,
		label("Time Machine:") + " " + m.tmStatus + "    " +
			label("Refresh:") + fmt.Sprintf(" %ds    ", int(m.cfg.RefreshInterval.Seconds())) +
			label("Last:") + " " + lastRefresh,
		m.formatAutoStatus(),
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, cw, "")
	}

	body := strings.Join(lines, "\n")

	if m.flash.active && (m.flash.gainPanel == panelInfo || m.flash.losePanel == panelInfo) {
		gaining := m.flash.gainPanel == panelInfo
		return renderFlashBorders(body, dot+" ", "snappy", spinnerSuffix, cw, m.flash, gaining, m.styles)
	}

	titleStyle := m.styles.sectionTitle
	if m.focusPanel != panelInfo {
		titleStyle = m.styles.sectionTitleDim
	}
	title := dot + " " + titleStyle.Render("snappy") + spinnerSuffix

	style := m.styles.section
	if m.focusPanel == panelInfo {
		style = m.styles.sectionFocus
	}

	rendered := style.Width(cw + 4).Render(body)

	borderFg := lipgloss.NewStyle().Foreground(style.GetBorderTopForeground())
	return borderTitle(rendered, title, borderFg)
}

func (m Model) renderSnapshotPanel(width int) string {
	cw := contentWidth(width)
	count := len(m.snapshots)
	titleLabel := fmt.Sprintf("local snapshots (%d)", count)

	if m.flash.active && (m.flash.gainPanel == panelSnap || m.flash.losePanel == panelSnap) {
		gaining := m.flash.gainPanel == panelSnap
		return renderFlashBorders(m.snapTable.View(), "", titleLabel, "", cw, m.flash, gaining, m.styles)
	}

	titleStyle := m.styles.sectionTitle
	if m.focusPanel != panelSnap {
		titleStyle = m.styles.sectionTitleDim
	}
	title := titleStyle.Render(titleLabel)

	style := m.styles.section
	if m.focusPanel == panelSnap {
		style = m.styles.sectionFocus
	}

	rendered := style.Width(cw + 4).Render(m.snapTable.View())
	borderFg := lipgloss.NewStyle().Foreground(style.GetBorderTopForeground())
	return borderTitle(rendered, title, borderFg)
}

func (m Model) renderLogPanel(width int) string {
	cw := contentWidth(width)
	titleLabel := "recent log"

	if m.flash.active && (m.flash.gainPanel == panelLog || m.flash.losePanel == panelLog) {
		gaining := m.flash.gainPanel == panelLog
		return renderFlashBorders(m.logView.View(), "", titleLabel, "", cw, m.flash, gaining, m.styles)
	}

	titleStyle := m.styles.sectionTitle
	if m.focusPanel != panelLog {
		titleStyle = m.styles.sectionTitleDim
	}
	title := titleStyle.Render(titleLabel)

	style := m.styles.section
	if m.focusPanel == panelLog {
		style = m.styles.sectionFocus
	}

	rendered := style.Width(cw + 4).Render(m.logView.View())
	borderFg := lipgloss.NewStyle().Foreground(style.GetBorderTopForeground())
	return borderTitle(rendered, title, borderFg)
}

func (m Model) renderHelpBar(_ int) string {
	return m.styles.helpBar.Render(m.help.View(m.keys))
}

func (m Model) formatAutoStatus() string {
	label := m.styles.infoLabel.Render
	if m.daemonActive {
		return label("Auto-snapshot:") + " " + indicatorOn + " " +
			m.styles.textCyan.Render("service") +
			fmt.Sprintf("    %s %ds    %s >%dm to %ds",
				label("every"), int(m.cfg.AutoSnapshotInterval.Seconds()),
				label("thin"), int(m.cfg.ThinAgeThreshold.Minutes()),
				int(m.cfg.ThinCadence.Seconds()),
			)
	}
	if m.auto.Enabled() {
		now := m.now()
		nextIn := int(m.auto.NextIn(now).Seconds())
		return label("Auto-snapshot:") + " " + indicatorOn + " " +
			m.styles.statusOn.Render("on") +
			fmt.Sprintf("    %s %ds    %s %ds    %s >%dm to %ds",
				label("every"), int(m.auto.Interval().Seconds()),
				label("next in"), nextIn,
				label("thin"), int(m.auto.ThinAge().Minutes()),
				int(m.auto.ThinCadence().Seconds()),
			)
	}
	return label("Auto-snapshot:") + " " + indicatorOff + " " +
		m.styles.statusOff.Render("off")
}

// borderTitle replaces the top border of a lipgloss-rendered bordered box
// with a new top line that embeds the given title, centered:
//
//	╭──────── title ────────╮
func borderTitle(rendered, title string, borderFg lipgloss.Style) string {
	lines := strings.SplitN(rendered, "\n", 2)
	if len(lines) < 2 {
		return rendered
	}

	topWidth := lipgloss.Width(lines[0])

	// Truncate the title if it would exceed the available border width.
	// Available space = topWidth - 4 (TopLeft + space + space + TopRight).
	if maxTitle := topWidth - 4; maxTitle > 0 {
		title = ansi.Truncate(title, maxTitle, "")
	}
	titleWidth := lipgloss.Width(title)

	border := lipgloss.RoundedBorder()

	// TopLeft(1) + leftFill + space(1) + title + space(1) + rightFill + TopRight(1)
	totalFill := max(topWidth-titleWidth-4, 0)
	leftFill := totalFill / 2
	rightFill := totalFill - leftFill

	newTop := borderFg.Render(border.TopLeft+strings.Repeat(border.Top, leftFill)+" ") +
		title +
		borderFg.Render(" "+strings.Repeat(border.Top, rightFill)+border.TopRight)

	return newTop + "\n" + lines[1]
}
