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

	titleBar := m.renderTitleBar(w)
	infoPanel := m.renderInfoPanel(w)
	snapPanel := m.renderSnapshotPanel(w)
	logPanel := m.renderLogPanel(w)
	helpBar := m.renderHelpBar(w)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		infoPanel,
		snapPanel,
		logPanel,
		helpBar,
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) renderTitleBar(width int) string {
	dot := indicatorOff
	if m.auto.Enabled() {
		dot = indicatorOn
	}
	if m.loading {
		dot = m.styles.textYellow.Render(dot)
	} else {
		dot = m.styles.textGreen.Render(dot)
	}

	title := dot + " SNAPPY"

	switch {
	case m.snapshotting:
		title += "  Snapshotting " + m.spinner.View()
	case m.thinning:
		title += "  Thinning " + m.spinner.View()
	case m.loading:
		title += "  Refreshing " + m.spinner.View()
	}

	return m.styles.titleBar.Width(width).Render(title)
}

func (m Model) renderInfoPanel(width int) string {
	cw := contentWidth(width)

	lastRefresh := "never"
	if !m.lastRefresh.IsZero() {
		lastRefresh = m.lastRefresh.Format("2006-01-02T15:04:05")
	}

	lines := []string{
		fmt.Sprintf("Volume: %s    Refresh: %ds    Last: %s",
			m.cfg.MountPoint, int(m.cfg.RefreshInterval.Seconds()), lastRefresh),
		fmt.Sprintf("Time Machine: %s", m.tmStatus),
	}

	if m.apfsVolume != "" {
		lines = append(lines, fmt.Sprintf("APFS Volume: %s    Other snapshots: %d",
			m.apfsVolume, m.otherSnapCount))
	}

	diskInfo := m.diskInfo
	if diskInfo == "" {
		diskInfo = "unavailable"
	}
	lines = append(lines,
		fmt.Sprintf("Disk: %s", diskInfo),
		m.formatAutoStatus(),
	)

	body := strings.Join(lines, "\n")
	return m.styles.section.Width(cw).Render(body)
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
	if m.auto.Enabled() {
		now := m.now()
		nextIn := int(m.auto.NextIn(now).Seconds())
		return fmt.Sprintf("Auto-snapshot: %s %s    every %ds    next in %ds    thin >%dm to %ds",
			indicatorOn,
			m.styles.statusOn.Render("on"),
			int(m.auto.Interval().Seconds()),
			nextIn,
			int(m.auto.ThinAge().Minutes()),
			int(m.auto.ThinCadence().Seconds()),
		)
	}
	return fmt.Sprintf("Auto-snapshot: %s %s",
		indicatorOff,
		m.styles.statusOff.Render("off"),
	)
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
