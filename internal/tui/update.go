package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/snapshot"
)

// Update handles incoming messages and returns the updated model and
// any commands to execute.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.BackgroundColorMsg:
		m.hasDarkBG = msg.IsDark()
		m.styles = newModelStyles(m.hasDarkBG)
		m.help.Styles = helpStyles(m.styles)
		m.spinner.Style = m.styles.spinnerStyle
		m.snapTable.SetStyles(m.styles.tableStyles)
		m.updateSnapViewContent()
		m.updateLogViewContent()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case UITickMsg:
		m.updateSnapViewContent()
		return m, uiTick()

	case RefreshTickMsg:
		return m.handleTick()

	case RefreshResultMsg:
		return m.handleRefreshResult(msg)

	case SnapshotCreatedMsg:
		return m.handleSnapshotCreated(msg)

	case ThinResultMsg:
		return m.handleThinResult(msg)
	}

	return m, nil
}

func helpStyles(s modelStyles) help.Styles {
	keyStyle := s.helpBar.Bold(true)
	descStyle := s.helpBar
	sepStyle := s.helpBar.Faint(true)
	return help.Styles{
		ShortKey:       keyStyle,
		ShortDesc:      descStyle,
		ShortSeparator: sepStyle,
		Ellipsis:       sepStyle,
		FullKey:        keyStyle,
		FullDesc:       descStyle,
		FullSeparator:  sepStyle,
	}
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	cw := contentWidth(m.width)
	m.help.SetWidth(m.width)

	// Fixed-height rows: info panel + snap/log borders + help bar.
	// Info panel: 3 body lines + 2 borders = 5.
	// Snap panel: 2 borders (table header is inside SetHeight).
	// Log panel: 2 borders.
	// Help bar: 1.
	const (
		infoHeight  = 5
		fixedHeight = infoHeight + 2 + 2 + 1 // 10
	)
	snapH, logH := flexPanelHeights(m.height, fixedHeight)

	m.snapPanelY = infoHeight
	m.logPanelY = infoHeight + 2 + snapH

	m.snapTable.SetWidth(cw)
	m.snapTable.SetHeight(snapH)
	m.logView.SetWidth(cw)
	m.logView.SetHeight(logH)

	m.updateSnapViewContent()
	m.updateLogViewContent()

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Snapshot):
		if m.snapshotting {
			return m, nil
		}
		m.snapshotting = true
		m.loading = true
		m.log.Log(logger.Info, "Creating snapshot...")
		m.updateLogViewContent()
		return m, tea.Batch(doCreateSnapshot(m.runner), m.spinner.Tick)

	case key.Matches(msg, m.keys.Refresh):
		clear(m.thinPinned)
		if m.refreshing {
			m.refreshPending = true
			return m, nil
		}
		m.refreshing = true
		m.loading = true
		return m, tea.Batch(doRefresh(m.runner, m.cfg, m.apfsVolume), m.spinner.Tick)

	case key.Matches(msg, m.keys.AutoSnap):
		now := m.now()
		enabled := m.auto.Toggle(now)
		if enabled {
			clear(m.thinPinned)
			m.log.Log(logger.Info, fmt.Sprintf(
				"Auto-snapshots enabled (every %ds, thin >%ds to %ds)",
				int(m.auto.Interval().Seconds()),
				int(m.auto.ThinAge().Seconds()),
				int(m.auto.ThinCadence().Seconds()),
			))
		} else {
			m.log.Log(logger.Info, "Auto-snapshots disabled")
		}
		m.updateLogViewContent()
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		m.log.Log(logger.Info, "Shutting down")
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Tab):
		m.setFocusPanel((m.focusPanel + 1) % 3)
		return m, nil

	case key.Matches(msg, m.keys.ScrollUp, m.keys.ScrollDown):
		return m.handleScroll(msg)
	}

	return m, nil
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	switch {
	case msg.Y >= m.logPanelY:
		m.setFocusPanel(panelLog)
		// Translate click Y to a log cursor position.
		// +1 for the border top line, +1 for any padding offset.
		row := msg.Y - m.logPanelY - 1 + m.logView.YOffset()
		if row >= 0 && row < m.logCount {
			m.logCursor = row
			m.updateLogViewContent()
		}
	case msg.Y >= m.snapPanelY:
		m.setFocusPanel(panelSnap)
		// Translate click Y to a table row. The table has a 1-line
		// header inside the bordered panel (+1 border top, +1 header).
		// The table's internal start = max(cursor - viewportHeight, 0).
		vh := m.snapTable.Height()
		start := max(m.snapTable.Cursor()-vh, 0)
		row := start + msg.Y - m.snapPanelY - 2
		if row >= 0 && row < len(m.snapshots) {
			m.snapTable.SetCursor(row)
		}
	default:
		m.setFocusPanel(panelInfo)
	}
	return m, nil
}

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if msg.Y >= m.logPanelY {
		m.setFocusPanel(panelLog)
		switch msg.Button {
		case tea.MouseWheelUp:
			m.moveLogCursor(-1)
		case tea.MouseWheelDown:
			m.moveLogCursor(1)
		}
		return m, nil
	}
	if msg.Y >= m.snapPanelY {
		m.setFocusPanel(panelSnap)
		switch msg.Button {
		case tea.MouseWheelUp:
			m.snapTable.MoveUp(1)
		case tea.MouseWheelDown:
			m.snapTable.MoveDown(1)
		}
	}
	return m, nil
}

func (m Model) handleScroll(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.focusPanel {
	case panelLog:
		if key.Matches(msg, m.keys.ScrollUp) {
			m.moveLogCursor(-1)
		} else {
			m.moveLogCursor(1)
		}
		return m, nil
	case panelSnap:
		var cmd tea.Cmd
		m.snapTable, cmd = m.snapTable.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) setFocusPanel(panel int) {
	m.focusPanel = panel
	if panel == panelSnap {
		m.snapTable.Focus()
	} else {
		m.snapTable.Blur()
	}
}

func (m *Model) moveLogCursor(delta int) {
	if m.logCount == 0 {
		return
	}
	m.logCursor = max(min(m.logCursor+delta, m.logCount-1), 0)
	m.updateLogViewContent()

	// Keep cursor visible in viewport.
	h := m.logView.Height()
	offset := m.logView.YOffset()
	if m.logCursor < offset {
		m.logView.SetYOffset(m.logCursor)
	} else if m.logCursor >= offset+h {
		m.logView.SetYOffset(m.logCursor - h + 1)
	}
}

func (m Model) handleTick() (tea.Model, tea.Cmd) {
	now := m.now()

	var cmds []tea.Cmd

	snapshotDue := m.auto.ShouldSnapshot(now) && !m.snapshotting
	if snapshotDue {
		m.snapshotting = true
		m.loading = true
		m.auto.RecordSnapshot(now)
		m.log.Log(logger.Auto, "Creating auto-snapshot...")
		m.updateLogViewContent()
		cmds = append(cmds, doCreateSnapshot(m.runner), m.spinner.Tick)
	}

	// Skip refresh when an auto-snapshot is in flight; SnapshotCreatedMsg
	// will trigger a post-creation refresh, avoiding a race where refresh
	// fetches the pre-snapshot list.
	if !snapshotDue && !m.refreshing {
		m.refreshing = true
		cmds = append(cmds, doRefresh(m.runner, m.cfg, m.apfsVolume))
	}
	cmds = append(cmds, refreshTick(m.cfg.RefreshInterval))

	return m, tea.Batch(cmds...)
}

func (m Model) handleRefreshResult(msg RefreshResultMsg) (tea.Model, tea.Cmd) {
	m.refreshing = false
	if !m.thinning && !m.snapshotting {
		m.loading = false
	}
	m.tmStatus = msg.TMStatus

	if msg.APFSInfo.Volume != "" {
		m.apfsVolume = msg.APFSInfo.Volume
		if msg.APFSInfo.OtherSnapCount > 0 {
			m.log.Log(logger.Info, fmt.Sprintf("Non-TM snapshots on %s: %d",
				msg.APFSInfo.Volume, msg.APFSInfo.OtherSnapCount))
		}
	}

	if msg.DiskErr {
		m.diskInfo = "unavailable"
	} else {
		m.diskInfo = msg.DiskInfo.String()
	}

	if msg.APFSErr != nil {
		m.log.Log(logger.Error, fmt.Sprintf("APFS details unavailable: %v", msg.APFSErr))
	}

	if msg.SnapshotErr != nil {
		m.log.Log(logger.Error, fmt.Sprintf("Failed to list snapshots: %v", msg.SnapshotErr))
		m.refreshPending = false
		m.updateLogViewContent()
		return m, nil
	}

	m.lastRefresh = m.now()

	// Compute diff
	prev := m.snapshots
	m.prevSnapshots = prev
	m.snapshots = msg.Snapshots

	if len(prev) > 0 || len(msg.Snapshots) > 0 {
		diff := snapshot.ComputeDiff(prev, msg.Snapshots)

		for _, s := range diff.Added {
			m.log.Log(logger.Added, "Snapshot appeared: "+s.Date)
		}
		for _, s := range diff.Removed {
			m.log.Log(logger.Removed, "Snapshot disappeared: "+s.Date)
		}
	}

	m.log.Log(logger.Info, fmt.Sprintf(
		"Refresh: %d snapshots, disk %s",
		len(m.snapshots), m.diskInfo,
	))

	m.updateSnapViewContent()
	m.updateLogViewContent()

	var cmds []tea.Cmd

	// If a refresh was requested while this one was in flight, re-refresh.
	if m.refreshPending {
		m.refreshPending = false
		m.refreshing = true
		cmds = append(cmds, doRefresh(m.runner, m.cfg, m.apfsVolume))
	}

	// Check thinning (skip if already in flight).
	cmds = m.maybeThin(cmds)

	return m, tea.Batch(cmds...)
}

func (m *Model) maybeThin(cmds []tea.Cmd) []tea.Cmd {
	if m.thinning {
		return cmds
	}
	targets := m.auto.ComputeThinTargets(m.snapshots, m.now(), m.thinPinned)
	// Belt-and-suspenders: filter out any pinned dates that slipped
	// through (ComputeThinTargets already skips them).
	filtered := targets[:0]
	for _, d := range targets {
		if _, ok := m.thinPinned[d]; !ok {
			filtered = append(filtered, d)
		}
	}
	if len(filtered) > 0 {
		m.thinning = true
		m.loading = true
		cmds = append(cmds, doThinSnapshots(m.runner, filtered), m.spinner.Tick)
	}
	return cmds
}

func (m Model) handleSnapshotCreated(msg SnapshotCreatedMsg) (tea.Model, tea.Cmd) {
	m.snapshotting = false
	if !m.thinning {
		m.loading = false
	}
	switch {
	case msg.Err != nil:
		m.log.Log(logger.Error, fmt.Sprintf("Failed to create snapshot: %v", msg.Err))
	case msg.Date != "":
		m.log.Log(logger.Created, "Snapshot created: "+msg.Date)
	default:
		m.log.Log(logger.Created, "Snapshot created")
	}

	m.updateLogViewContent()

	if m.refreshing {
		m.refreshPending = true
		return m, nil
	}
	m.refreshing = true
	return m, doRefresh(m.runner, m.cfg, m.apfsVolume)
}

func (m Model) handleThinResult(msg ThinResultMsg) (tea.Model, tea.Cmd) {
	m.thinning = false
	if !m.snapshotting {
		m.loading = false
	}
	if msg.Deleted > 0 {
		m.log.Log(logger.Thinned, fmt.Sprintf(
			"Thinned %d snapshot(s) older than %dm to %ds cadence",
			msg.Deleted,
			int(m.auto.ThinAge().Minutes()),
			int(m.auto.ThinCadence().Seconds()),
		))
	}

	if msg.Err != nil {
		// Record which dates failed so we skip them next cycle.
		for _, d := range msg.FailedDates {
			m.thinPinned[d] = struct{}{}
		}
		m.log.Log(logger.Error, fmt.Sprintf("Thinning error: %v", msg.Err))
	} else {
		// Full success: conditions may have changed, clear pinned set.
		clear(m.thinPinned)
	}

	m.updateLogViewContent()

	// Only trigger a refresh when at least one deletion succeeded.
	// When all deletions failed, the same targets would reappear
	// immediately, causing a tight retry loop. The next regular tick
	// will refresh instead.
	if msg.Deleted == 0 {
		return m, nil
	}

	if m.refreshing {
		m.refreshPending = true
		return m, nil
	}
	m.refreshing = true
	return m, doRefresh(m.runner, m.cfg, m.apfsVolume)
}

// updateSnapViewContent rebuilds columns and rows on the snapshot table.
// Snapshots are listed newest first so both panels scroll the same direction.
func (m *Model) updateSnapViewContent() {
	cols := m.snapTableColumns()
	m.snapTable.SetColumns(cols)

	count := len(m.snapshots)
	if count == 0 {
		row := make(table.Row, len(cols))
		row[0] = "(none, press 's' to create the first snapshot)"
		m.snapTable.SetRows([]table.Row{row})
		return
	}

	rows := make([]table.Row, 0, count)
	now := m.now()
	for i := count - 1; i >= 0; i-- {
		snap := m.snapshots[i]
		date := snap.Time.Format("2006-01-02 15:04:05")
		age := snapshot.FormatRelativeTime(snap.Time, now)

		var xid, uuid, status string
		if snap.UUID != "" {
			xid = fmt.Sprintf("%d", snap.XID)
			uuid = snap.UUID
			if snap.LimitsShrink {
				status = indicatorWarning + " limits shrink"
			}
		}
		rows = append(rows, table.Row{date, age, xid, uuid, status})
	}
	m.snapTable.SetRows(rows)
}

// snapTableColumns returns the column definitions for the snapshot table,
// sized to fit the current terminal width. UUID acts as the flex column,
// absorbing any remaining width so the table fills the panel.
func (m *Model) snapTableColumns() []table.Column {
	// Column.Width is the text content width. The Cell/Header styles add
	// Padding(0,1) which contributes 2 extra rendered chars per column
	// (1 left + 1 right).
	const (
		colPad       = 3 // rendered padding per column (right only)
		ncols        = 5
		dateWidth    = 19 // "2006-01-02 15:04:05"
		ageWidth     = 5
		xidWidth     = 7
		uuidMinWidth = 9  // first UUID segment + ellipsis
		uuidMaxWidth = 36 // full UUID
		statusMin    = 20
	)

	tw := m.snapTable.Width()
	fixedWidth := dateWidth + ageWidth + xidWidth + ncols*colPad
	remaining := tw - fixedWidth
	uuidWidth := min(max(remaining-statusMin, uuidMinWidth), uuidMaxWidth)
	statusWidth := max(remaining-uuidWidth, 0)

	return []table.Column{
		{Title: "DATE", Width: dateWidth},
		{Title: "AGE", Width: ageWidth},
		{Title: "XID", Width: xidWidth},
		{Title: "UUID", Width: uuidWidth},
		{Title: "STATUS", Width: statusWidth},
	}
}

// updateLogViewContent rebuilds the log viewport content.
// Entries are shown newest first so both panels scroll the same direction.
// The line at logCursor is rendered bold.
func (m *Model) updateLogViewContent() {
	entries := m.log.Entries()
	m.logCount = len(entries)
	if m.logCursor >= m.logCount {
		m.logCursor = max(m.logCount-1, 0)
	}

	if m.logCount == 0 {
		m.logView.SetContent(m.styles.textDim.Render("(no log entries yet)"))
		return
	}

	w := m.logView.Width()
	var b strings.Builder
	displayIdx := 0
	for i := len(entries) - 1; i >= 0; i-- {
		if displayIdx > 0 {
			b.WriteByte('\n')
		}
		e := entries[i]
		line := fmt.Sprintf("%-8s   %-7s   %s",
			e.Timestamp.Format("15:04:05"),
			string(e.Type),
			e.Message,
		)
		style := logEntryStyle(m.styles, e.Type)
		if displayIdx == m.logCursor {
			style = style.Bold(true)
		}
		b.WriteString(ansi.Truncate(style.Render(line), w, ""))
		displayIdx++
	}
	m.logView.SetContent(b.String())
}

// logEntryStyle returns the lipgloss style for the given log entry type.
func logEntryStyle(s modelStyles, t logger.EventType) lipgloss.Style {
	switch t {
	case logger.Error:
		return s.textRed
	case logger.Created, logger.Added:
		return s.textGreen
	case logger.Removed, logger.Thinned:
		return s.textYellow
	case logger.Auto:
		return s.textCyan
	case logger.Startup:
		return s.textMagenta
	default:
		return lipgloss.NewStyle()
	}
}
