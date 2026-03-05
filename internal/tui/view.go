package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/snapshot"
)

// View renders the full TUI screen as a tea.View.
func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

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
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
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

	title := dot + " " + lipgloss.NewStyle().Bold(true).Render("snappy")

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
	lines := []string{
		label("Volume:") + " " + m.volumeName + "    " + label("Disk:") + " " + diskInfo,
		label("Time Machine:") + " " + m.tmStatus + "    " +
			label("Refresh:") + fmt.Sprintf(" %ds    ", int(m.cfg.RefreshInterval.Seconds())) +
			label("Last:") + " " + lastRefresh,
		m.formatAutoStatus(),
	}

	body := strings.Join(lines, "\n")
	rendered := m.styles.section.Width(cw).Render(body)

	borderFg := lipgloss.NewStyle().Foreground(m.styles.section.GetBorderTopForeground())
	return borderTitle(rendered, title, borderFg)
}

func (m Model) renderSnapshotPanel(width int) string {
	cw := contentWidth(width)
	count := len(m.snapshots)

	diffSummary := ""
	if m.diffAdded > 0 || m.diffRemoved > 0 {
		diffSummary = fmt.Sprintf("  [+%d added, %d removed]", m.diffAdded, m.diffRemoved)
	}

	title := m.styles.sectionTitle.Render(fmt.Sprintf("LOCAL SNAPSHOTS (%d)", count)) + diffSummary

	body := title + "\n" + m.snapView.View()

	style := m.styles.section
	if !m.focusLog {
		style = m.styles.sectionFocus
	}
	return style.Width(cw).Render(body)
}

func (m Model) renderLogPanel(width int) string {
	cw := contentWidth(width)

	title := m.styles.sectionTitle.Render("RECENT LOG")
	body := title + "\n" + m.logView.View()

	style := m.styles.section
	if m.focusLog {
		style = m.styles.sectionFocus
	}
	return style.Width(cw).Render(body)
}

func (m Model) renderHelpBar(_ int) string {
	return m.styles.helpBar.Render(m.help.View(m.keys))
}

func (m Model) formatAutoStatus() string {
	label := m.styles.infoLabel.Render
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

func (m Model) formatSnapshotLine(i, count int) string {
	snap := m.snapshots[i]
	now := m.now()
	relative := snapshot.FormatRelativeTime(snap.Time, now)

	number := m.styles.snapNumber.Render(fmt.Sprintf("%2d.", count-i))
	timeStr := m.styles.textDim.Render(fmt.Sprintf("(%s)", relative))

	details := ""
	if m.apfsVolume != "" && snap.UUID != "" {
		flags := indicatorPurge + " purgeable"
		if !snap.Purgeable {
			flags = m.styles.textYellow.Render(indicatorPinned + " pinned")
		}
		if snap.LimitsShrink {
			flags += "  " + m.styles.textRed.Render(indicatorWarning+" limits shrink")
		}
		details = fmt.Sprintf("   %s   %s", snap.UUID, flags)
	}

	return fmt.Sprintf("%s  %s   %s%s", number, snap.Date, timeStr, details)
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

func (m Model) colorizeLogEntry(entry logger.Entry) string {
	switch entry.Type {
	case logger.Error:
		return m.styles.textRed.Render(entry.Formatted)
	case logger.Created, logger.Added:
		return m.styles.textGreen.Render(entry.Formatted)
	case logger.Removed, logger.Thinned:
		return m.styles.textYellow.Render(entry.Formatted)
	case logger.Auto:
		return m.styles.textCyan.Render(entry.Formatted)
	case logger.Startup:
		return m.styles.textMagenta.Render(entry.Formatted)
	default:
		return entry.Formatted
	}
}
