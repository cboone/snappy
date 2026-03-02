package tui

import (
	"fmt"
	"strings"

	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/snapshot"
)

const maxLogLines = 8

// View renders the full TUI screen.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	m.renderHeader(&b)
	m.renderSnapshotList(&b)
	m.renderRecentLog(&b)
	m.renderControls(&b)

	return b.String()
}

func (m Model) renderHeader(b *strings.Builder) {
	fmt.Fprintln(b, separator)
	fmt.Fprintf(b, "  %s\n", titleStyle.Render(fmt.Sprintf("SNAPPY v%s -- Time Machine Local Snapshot Manager", m.version)))
	fmt.Fprintln(b, separator)

	lastRefresh := "never"
	if !m.lastRefresh.IsZero() {
		lastRefresh = m.lastRefresh.Format("2006-01-02T15:04:05")
	}
	fmt.Fprintf(b, "  Volume: %s    |  Refresh: %ds  |  Last: %s\n",
		m.cfg.MountPoint, int(m.cfg.RefreshInterval.Seconds()), lastRefresh)

	fmt.Fprintf(b, "  Time Machine: %s\n", m.tmStatus)

	if m.apfsVolume != "" {
		fmt.Fprintf(b, "  APFS Volume: %s  |  Other snapshots: %d (non-Time Machine)\n",
			m.apfsVolume, m.otherSnapCount)
	}

	diskInfo := m.diskInfo
	if diskInfo == "" {
		diskInfo = "unavailable"
	}
	fmt.Fprintf(b, "  Disk: %s\n", diskInfo)

	m.renderAutoStatus(b)

	fmt.Fprintln(b, separator)
}

func (m Model) renderAutoStatus(b *strings.Builder) {
	if m.auto.Enabled() {
		now := m.now()
		nextIn := int(m.auto.NextIn(now).Seconds())
		fmt.Fprintf(b, "  Auto-snapshot: %s  |  every %ds  |  next in %ds  |  thin >%dm to %ds\n",
			green.Render("on"),
			int(m.auto.Interval().Seconds()),
			nextIn,
			int(m.auto.ThinAge().Minutes()),
			int(m.auto.ThinCadence().Seconds()),
		)
	} else {
		fmt.Fprintf(b, "  Auto-snapshot: %s\n", red.Render("off"))
	}
}

func (m Model) renderSnapshotList(b *strings.Builder) {
	count := len(m.snapshots)

	diffSummary := ""
	if m.diffAdded > 0 || m.diffRemoved > 0 {
		diffSummary = fmt.Sprintf("    [+%d added, %d removed]", m.diffAdded, m.diffRemoved)
	}

	fmt.Fprintln(b)
	fmt.Fprintf(b, "  %s%s\n", bold.Render(fmt.Sprintf("LOCAL SNAPSHOTS (%d)", count)), diffSummary)
	fmt.Fprintf(b, "  %s\n", thinSeparator)

	if count == 0 {
		fmt.Fprintf(b, "  %s\n", dim.Render("(none -- press 's' to create the first snapshot)"))
		return
	}

	if count <= 4 {
		for i := count - 1; i >= 0; i-- {
			m.renderSnapshotLine(b, i, count)
		}
		return
	}

	// Bookend: 2 newest, ellipsis, 2 oldest
	m.renderSnapshotLine(b, count-1, count)
	m.renderSnapshotLine(b, count-2, count)

	hidden := count - 4
	fmt.Fprintf(b, "  %s\n", dim.Render(fmt.Sprintf("  ... and %d more ...", hidden)))

	m.renderSnapshotLine(b, 1, count)
	m.renderSnapshotLine(b, 0, count)
}

func (m Model) renderSnapshotLine(b *strings.Builder, i, count int) {
	snap := m.snapshots[i]
	now := m.now()
	relative := snapshot.FormatRelativeTime(snap.Time, now)

	number := green.Render(fmt.Sprintf("%2d.", count-i))
	timeStr := dim.Render(fmt.Sprintf("(%s)", relative))

	details := ""
	if m.apfsVolume != "" && snap.UUID != "" {
		flags := "purgeable"
		if !snap.Purgeable {
			flags = yellow.Render("pinned")
		}
		if snap.LimitsShrink {
			flags += "  " + red.Render("limits shrink")
		}
		details = fmt.Sprintf("   %s   %s", snap.UUID, flags)
	}

	fmt.Fprintf(b, "  %s  %s   %s%s\n", number, snap.Date, timeStr, details)
}

func (m Model) renderRecentLog(b *strings.Builder) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, separator)
	fmt.Fprintf(b, "  %s\n", bold.Render("RECENT LOG"))
	fmt.Fprintln(b, separator)

	entries := m.log.Entries()
	if len(entries) == 0 {
		fmt.Fprintf(b, "  %s\n", dim.Render("(no log entries yet)"))
		return
	}

	start := 0
	if len(entries) > maxLogLines {
		start = len(entries) - maxLogLines
	}

	// Newest first
	for i := len(entries) - 1; i >= start; i-- {
		entry := entries[i]
		styled := colorizeLogEntry(entry)
		fmt.Fprintf(b, "  %s\n", styled)
	}
}

func colorizeLogEntry(entry logger.Entry) string {
	switch entry.Type {
	case logger.Error:
		return red.Render(entry.Formatted)
	case logger.Created, logger.Added:
		return green.Render(entry.Formatted)
	case logger.Removed, logger.Thinned:
		return yellow.Render(entry.Formatted)
	case logger.Auto:
		return cyan.Render(entry.Formatted)
	case logger.Startup:
		return magenta.Render(entry.Formatted)
	default:
		return entry.Formatted
	}
}

func (m Model) renderControls(b *strings.Builder) {
	fmt.Fprintln(b)
	fmt.Fprintln(b, separator)
	fmt.Fprintf(b, "  %s Snapshot   %s Refresh   %s Auto-snap   %s Quit\n",
		keyStyle.Render("[s]"),
		keyStyle.Render("[r]"),
		keyStyle.Render("[a]"),
		keyStyle.Render("[q]"),
	)
	fmt.Fprintln(b, separator)
}
