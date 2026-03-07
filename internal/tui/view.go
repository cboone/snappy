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

func (m Model) renderInfoPanel(width int) string {
	cw := contentWidth(width)

	// Build the title string for embedding in the border.
	dot := indicatorOff
	if m.auto.Enabled() {
		dot = indicatorOn
	}
	if m.loading {
		dot = m.styles.textYellow.Render(dot)
	} else {
		dot = m.styles.textGreen.Render(dot)
	}

	titleStyle := m.styles.sectionTitle
	if m.focusPanel != panelInfo {
		titleStyle = m.styles.sectionTitleDim
	}
	title := dot + " " + titleStyle.Render("snappy")

	switch {
	case m.snapshotting:
		title += "  Snapshotting " + m.spinner.View()
	case m.thinning:
		title += "  Thinning " + m.spinner.View()
	case m.loading:
		title += "  Refreshing " + m.spinner.View()
	}

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

	style := m.styles.section
	if m.focusPanel == panelInfo {
		style = m.styles.sectionFocus
	}

	body := strings.Join(lines, "\n")
	rendered := style.Width(cw + 4).Render(body)

	borderFg := lipgloss.NewStyle().Foreground(style.GetBorderTopForeground())
	return borderTitle(rendered, title, borderFg)
}

func (m Model) renderSnapshotPanel(width int) string {
	sw := contentWidth(width) + 4
	count := len(m.snapshots)

	titleStyle := m.styles.sectionTitle
	if m.focusPanel != panelSnap {
		titleStyle = m.styles.sectionTitleDim
	}
	title := titleStyle.Render(fmt.Sprintf("local snapshots (%d)", count))

	style := m.styles.section
	if m.focusPanel == panelSnap {
		style = m.styles.sectionFocus
	}

	header := m.snapHeaderLine
	bodyLines := m.snapBodyLines

	var clipped string
	if len(bodyLines) > 0 {
		end := min(m.snapScrollOffset+m.snapVisibleRows, len(bodyLines))
		start := min(m.snapScrollOffset, end)
		visible := make([]string, 0, m.snapVisibleRows)
		visible = append(visible, bodyLines[start:end]...)
		for len(visible) < m.snapVisibleRows {
			visible = append(visible, "")
		}
		clipped = header + "\n" + strings.Join(visible, "\n")
	} else {
		clipped = header
	}

	rendered := style.Width(sw).Render(clipped)
	borderFg := lipgloss.NewStyle().Foreground(style.GetBorderTopForeground())
	return borderTitle(rendered, title, borderFg)
}

func (m Model) renderLogPanel(width int) string {
	sw := contentWidth(width) + 4
	titleStyle := m.styles.sectionTitle
	if m.focusPanel != panelLog {
		titleStyle = m.styles.sectionTitleDim
	}
	title := titleStyle.Render("recent log")

	style := m.styles.section
	if m.focusPanel == panelLog {
		style = m.styles.sectionFocus
	}

	rendered := style.Width(sw).Render(m.logView.View())
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
