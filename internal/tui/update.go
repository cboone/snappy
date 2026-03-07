package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/service"
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
		return m.handleBackgroundColor(msg)

	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	case UITickMsg:
		return m.handleUITick()

	case RefreshTickMsg:
		return m.handleTick()

	case RefreshResultMsg:
		return m.handleRefreshResult(msg)

	case SnapshotCreatedMsg:
		return m.handleSnapshotCreated(msg)

	case FlashTickMsg:
		return m.handleFlashTick(msg)

	case ThinResultMsg:
		return m.handleThinResult(msg)

	case OpenLogDirResultMsg:
		if msg.Err != nil {
			m.log.Log(logger.LevelError, logger.CatOpen, fmt.Sprintf("Failed to open log directory: %v", msg.Err))
			m.updateLogViewContent()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleBackgroundColor(msg tea.BackgroundColorMsg) (tea.Model, tea.Cmd) {
	m.hasDarkBG = msg.IsDark()
	m.styles = newModelStyles(m.hasDarkBG)
	m.help.Styles = helpStyles(m.styles)
	m.spinner.Style = m.styles.spinnerStyle
	m.snapTable.SetStyles(m.styles.tableStyles)
	m.updateSnapViewContent()
	m.updateLogViewContent()
	return m, nil
}

func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleUITick() (tea.Model, tea.Cmd) {
	m.updateSnapAges()
	if m.auto.Enabled() || m.loading {
		return m, uiTick()
	}
	return m, nil
}

func helpStyles(s modelStyles) help.Styles {
	// Derive segment styles from a no-padding base so that padding is only
	// applied by the outer help bar renderer in renderHelpBar.
	base := lipgloss.NewStyle().
		Foreground(s.helpBar.GetForeground())
	keyStyle := base.Bold(true)
	descStyle := base
	sepStyle := base.Faint(true)
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
	// Snap panel: 2 borders (visible rows set via snapVisibleRows).
	// Log panel: 2 borders.
	// Help bar: 1.
	const (
		infoHeight  = 5
		fixedHeight = infoHeight + 2 + 2 + 1 // 10
	)
	snapH, logH := flexPanelHeights(m.height, fixedHeight)

	m.snapPanelY = infoHeight
	m.logPanelY = infoHeight + 2 + snapH
	m.helpBarY = m.logPanelY + 2 + logH

	m.snapTable.SetWidth(cw)
	m.snapVisibleRows = snapH - 1
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
		m.log.Log(logger.LevelInfo, logger.CatSnapshot, "Creating snapshot...")
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
		return m, tea.Batch(doRefresh(m.runner, m.apfsVolume, m.apfsContainer), m.spinner.Tick)

	case key.Matches(msg, m.keys.AutoSnap):
		if m.daemonActive {
			m.log.Log(logger.LevelInfo, logger.CatAuto, "Auto-snapshots managed by background service (snappy service stop to take over)")
			m.updateLogViewContent()
			return m, nil
		}
		now := m.now()
		enabled := m.auto.Toggle(now)
		if enabled {
			clear(m.thinPinned)
			m.log.Log(logger.LevelInfo, logger.CatAuto, fmt.Sprintf(
				"Auto-snapshots enabled (every %ds, thin >%ds to %ds)",
				int(m.auto.Interval().Seconds()),
				int(m.auto.ThinAge().Seconds()),
				int(m.auto.ThinCadence().Seconds()),
			))
			m.updateLogViewContent()
			return m, uiTick()
		}
		m.log.Log(logger.LevelInfo, logger.CatAuto, "Auto-snapshots disabled")
		m.updateLogViewContent()
		return m, nil

	case key.Matches(msg, m.keys.OpenLog):
		if m.cfg.LogDir == "" {
			m.log.Log(logger.LevelWarn, logger.CatOpen, "Log directory unavailable")
			m.updateLogViewContent()
			return m, nil
		}
		m.log.Log(logger.LevelInfo, logger.CatOpen, "Opening log directory...")
		m.updateLogViewContent()
		return m, doOpenLogDir(m.cfg.LogDir)

	case key.Matches(msg, m.keys.Quit):
		m.log.Log(logger.LevelInfo, logger.CatShutdown, "Shutting down")
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, m.keys.Tab):
		cmd := m.setFocusPanel((m.focusPanel + 1) % 3)
		return m, cmd

	case key.Matches(msg, m.keys.ShiftTab):
		m.setFocusPanel((m.focusPanel + 2) % 3)
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
	var cmd tea.Cmd
	switch {
	case msg.Y >= m.helpBarY:
		// Click is on the help bar; ignore.
	case msg.Y >= m.logPanelY:
		cmd = m.setFocusPanel(panelLog)
		// Only select entries for clicks inside the content area (not borders).
		contentY := msg.Y - m.logPanelY - 1
		if contentY >= 0 && contentY < m.logView.Height() {
			visualLine := contentY + m.logView.YOffset()
			entry := logEntryAtVisualLine(m.logEntryY, m.logTotalLines, visualLine)
			if entry >= 0 && entry < m.logCount {
				m.logCursor = entry
				m.updateLogViewContent()
			}
		}
	case msg.Y >= m.snapPanelY:
		cmd = m.setFocusPanel(panelSnap)
		// Translate click Y to a data row index: -1 border, -1 header.
		line := msg.Y - m.snapPanelY - 2
		if line >= 0 && line < m.snapVisibleRows {
			row := line + m.snapScrollOffset
			if row >= 0 && row < len(m.snapTable.Rows()) {
				m.snapTable.SetCursor(row)
				m.updateSnapRenderCache()
			}
		}
	default:
		cmd = m.setFocusPanel(panelInfo)
	}
	return m, cmd
}

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if msg.Y >= m.logPanelY && msg.Y < m.helpBarY {
		cmd := m.setFocusPanel(panelLog)
		switch msg.Button {
		case tea.MouseWheelUp:
			m.scrollLogView(-1)
		case tea.MouseWheelDown:
			m.scrollLogView(1)
		}
		return m, cmd
	}
	if msg.Y >= m.snapPanelY && msg.Y < m.logPanelY {
		cmd := m.setFocusPanel(panelSnap)
		switch msg.Button {
		case tea.MouseWheelUp:
			m.snapScrollOffset--
		case tea.MouseWheelDown:
			m.snapScrollOffset++
		}
		m.clampSnapScroll()
		return m, cmd
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
		m.ensureSnapCursorVisible()
		m.updateSnapRenderCache()
		return m, cmd
	}
	return m, nil
}

func (m *Model) setFocusPanel(panel int) tea.Cmd {
	if panel == m.focusPanel {
		return nil
	}
	prev := m.focusPanel
	nextID := m.flash.id + 1
	m.focusPanel = panel
	if panel == panelSnap {
		m.snapTable.Focus()
	} else {
		m.snapTable.Blur()
	}
	m.updateSnapRenderCache()
	m.flash = flashState{
		active:      true,
		gainPanel:   panel,
		losePanel:   prev,
		frame:       0,
		totalFrames: flashTotalFrames,
		id:          nextID,
	}
	return flashTick(nextID)
}

func (m Model) handleFlashTick(msg FlashTickMsg) (tea.Model, tea.Cmd) {
	if !m.flash.active || msg.ID != m.flash.id {
		return m, nil
	}
	m.flash.frame++
	if m.flash.frame >= m.flash.totalFrames {
		m.flash.active = false
		return m, nil
	}
	return m, flashTick(m.flash.id)
}

func (m *Model) moveLogCursor(delta int) {
	if m.logCount == 0 {
		return
	}
	m.logCursor = max(min(m.logCursor+delta, m.logCount-1), 0)
	m.updateLogViewContent()

	// Keep cursor visible in viewport using visual line offsets.
	if m.logCursor >= len(m.logEntryY) {
		return
	}
	entryY := m.logEntryY[m.logCursor]
	h := m.logView.Height()
	offset := m.logView.YOffset()
	if entryY < offset {
		m.logView.SetYOffset(entryY)
	} else if entryY >= offset+h {
		m.logView.SetYOffset(entryY - h + 1)
	}
}

func (m *Model) scrollLogView(delta int) {
	offset := m.logView.YOffset() + delta
	maxOffset := max(m.logTotalLines-m.logView.Height(), 0)
	m.logView.SetYOffset(max(min(offset, maxOffset), 0))
}

func (m *Model) clampSnapScroll() {
	maxOffset := max(len(m.snapTable.Rows())-m.snapVisibleRows, 0)
	m.snapScrollOffset = max(min(m.snapScrollOffset, maxOffset), 0)
}

func (m *Model) ensureSnapCursorVisible() {
	cursor := m.snapTable.Cursor()
	if cursor < m.snapScrollOffset {
		m.snapScrollOffset = cursor
	} else if cursor >= m.snapScrollOffset+m.snapVisibleRows {
		m.snapScrollOffset = cursor - m.snapVisibleRows + 1
	}
	m.clampSnapScroll()
}

func (m Model) handleTick() (tea.Model, tea.Cmd) {
	now := m.now()
	m.syncDaemonState(now)

	var cmds []tea.Cmd

	snapshotDue := m.auto.ShouldSnapshot(now) && !m.snapshotting
	if snapshotDue {
		m.snapshotting = true
		m.autoSnapshotting = true
		m.loading = true
		m.auto.RecordSnapshot(now)
		m.log.Log(logger.LevelInfo, logger.CatAuto, "Creating auto-snapshot...")
		m.updateLogViewContent()
		lockPath := service.DefaultLockPath(m.cfg.LogDir)
		cmds = append(cmds, doAutoCreateSnapshot(m.runner, lockPath), m.spinner.Tick)
	}

	// Skip refresh when an auto-snapshot is in flight; SnapshotCreatedMsg
	// will trigger a post-creation refresh, avoiding a race where refresh
	// fetches the pre-snapshot list.
	if !snapshotDue && !m.refreshing {
		m.refreshing = true
		cmds = append(cmds, doRefresh(m.runner, m.apfsVolume, m.apfsContainer))
	}
	cmds = append(cmds, refreshTick(m.cfg.RefreshInterval))

	return m, tea.Batch(cmds...)
}

func (m *Model) syncDaemonState(now time.Time) {
	if m.cfg.LogDir == "" {
		return
	}
	if m.autoSnapshotting {
		return
	}
	lockPath := service.DefaultLockPath(m.cfg.LogDir)
	lockHeld := service.IsHeld(lockPath)

	switch {
	case lockHeld && !m.daemonActive:
		m.daemonActive = true
		if m.auto.Enabled() {
			m.auto.Toggle(now)
		}
		m.log.Log(logger.LevelInfo, logger.CatAuto, "Background service detected; TUI auto-snapshots disabled")
		m.updateLogViewContent()

	case !lockHeld && m.daemonActive:
		m.daemonActive = false
		m.log.Log(logger.LevelInfo, logger.CatAuto, "Background service no longer detected; press 'a' to enable auto-snapshots")
		m.updateLogViewContent()
	}
}

func (m Model) handleRefreshResult(msg RefreshResultMsg) (tea.Model, tea.Cmd) {
	m.refreshing = false
	if !m.thinning && !m.snapshotting {
		m.loading = false
	}
	m.tmStatus = msg.TMStatus

	m.applyAPFSInfo(msg)
	m.applyTidemark(msg)
	m.applyDiskInfo(msg)

	if msg.APFSErr != nil {
		m.log.Log(logger.LevelError, logger.CatRefresh, fmt.Sprintf("APFS details unavailable: %v", msg.APFSErr))
	}

	if msg.SnapshotErr != nil {
		m.log.Log(logger.LevelError, logger.CatRefresh, fmt.Sprintf("Failed to list snapshots: %v", msg.SnapshotErr))
		m.refreshPending = false
		m.updateLogViewContent()
		return m, nil
	}

	m.lastRefresh = m.now()

	prev := m.snapshots
	m.prevSnapshots = prev
	m.snapshots = msg.Snapshots

	m.logDiffChanges(prev, msg.Snapshots)
	m.logRefreshSummary()

	m.updateSnapViewContent()
	m.updateLogViewContent()

	var cmds []tea.Cmd

	// If a refresh was requested while this one was in flight, re-refresh.
	if m.refreshPending {
		m.refreshPending = false
		m.refreshing = true
		cmds = append(cmds, doRefresh(m.runner, m.apfsVolume, m.apfsContainer))
	}

	// Check thinning (skip if already in flight).
	cmds = m.maybeThin(cmds)

	return m, tea.Batch(cmds...)
}

// applyAPFSInfo updates APFS volume state and logs non-TM snapshot count changes.
func (m *Model) applyAPFSInfo(msg RefreshResultMsg) {
	if msg.APFSInfo.Volume == "" {
		return
	}
	m.apfsVolume = msg.APFSInfo.Volume
	if msg.APFSInfo.OtherSnapCount > 0 && msg.APFSInfo.OtherSnapCount != m.lastOtherSnapCount {
		m.log.Log(logger.LevelInfo, logger.CatRefresh, fmt.Sprintf("Non-TM snapshots on %s: %d",
			msg.APFSInfo.Volume, msg.APFSInfo.OtherSnapCount))
	}
	m.lastOtherSnapCount = msg.APFSInfo.OtherSnapCount
}

// applyTidemark updates the tidemark display from a refresh result.
func (m *Model) applyTidemark(msg RefreshResultMsg) {
	if msg.TidemarkErr != nil {
		m.log.Log(logger.LevelWarn, logger.CatRefresh,
			"Tidemark fetch failed: "+msg.TidemarkErr.Error())
	}
	if msg.Tidemark > 0 {
		m.tidemark = platform.FormatBytes(msg.Tidemark)
	} else {
		m.tidemark = ""
	}
}

// applyDiskInfo updates the cached disk info string from a refresh result.
func (m *Model) applyDiskInfo(msg RefreshResultMsg) {
	if msg.DiskErr {
		m.diskInfo = "unavailable"
	} else {
		m.diskInfo = msg.DiskInfo.String()
	}
}

// logDiffChanges logs snapshot additions and removals between refreshes,
// suppressing duplicates from recent creates/thins.
func (m *Model) logDiffChanges(prev, current []snapshot.Snapshot) {
	if len(prev) == 0 && len(current) == 0 {
		if !m.hadFirstRefresh {
			m.hadFirstRefresh = true
		}
		return
	}

	diff := snapshot.ComputeDiff(prev, current)

	if !m.hadFirstRefresh && len(diff.Added) > 0 {
		foundCount := 0
		for _, s := range diff.Added {
			if _, ok := m.recentCreated[s.Date]; !ok {
				foundCount++
			}
		}
		if foundCount > 0 {
			m.log.Log(logger.LevelInfo, logger.CatFound, fmt.Sprintf(
				"Found %d existing snapshots", foundCount))
		}
	} else {
		for _, s := range diff.Added {
			if _, ok := m.recentCreated[s.Date]; ok {
				continue
			}
			m.log.Log(logger.LevelInfo, logger.CatAdded, "Snapshot appeared: "+s.Date)
		}
	}
	clear(m.recentCreated)

	for _, s := range diff.Removed {
		if _, ok := m.recentThinned[s.Date]; ok {
			continue
		}
		m.log.Log(logger.LevelInfo, logger.CatRemoved, "Snapshot disappeared: "+s.Date)
	}
	clear(m.recentThinned)

	m.hadFirstRefresh = true
}

// logRefreshSummary logs the refresh summary only when the content changes.
func (m *Model) logRefreshSummary() {
	summary := fmt.Sprintf("Refresh: %d snapshots, disk %s",
		len(m.snapshots), m.diskInfo)
	if summary != m.lastRefreshSummary {
		m.log.Log(logger.LevelInfo, logger.CatRefresh, summary)
		m.lastRefreshSummary = summary
	}
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
		m.log.Log(logger.LevelInfo, logger.CatAuto, fmt.Sprintf("Thinning %d snapshot(s)...", len(filtered)))
		m.updateLogViewContent()
		cmds = append(cmds, doThinSnapshots(m.runner, filtered), m.spinner.Tick)
	}
	return cmds
}

func (m Model) handleSnapshotCreated(msg SnapshotCreatedMsg) (tea.Model, tea.Cmd) {
	m.snapshotting = false
	m.autoSnapshotting = false
	if !m.thinning {
		m.loading = false
	}
	switch {
	case msg.Skipped:
		m.log.Log(logger.LevelInfo, logger.CatAuto, "Auto-snapshot skipped: daemon holds lock")
	case msg.Err != nil:
		m.log.Log(logger.LevelError, logger.CatSnapshot, fmt.Sprintf("Failed to create snapshot: %v", msg.Err))
	case msg.Date != "":
		m.recentCreated[msg.Date] = struct{}{}
		m.log.Log(logger.LevelInfo, logger.CatCreated, "Snapshot created: "+msg.Date)
	default:
		m.log.Log(logger.LevelInfo, logger.CatCreated, "Snapshot created")
	}

	m.updateLogViewContent()

	if m.refreshing {
		m.refreshPending = true
		return m, nil
	}
	m.refreshing = true
	return m, doRefresh(m.runner, m.apfsVolume, m.apfsContainer)
}

func (m Model) handleThinResult(msg ThinResultMsg) (tea.Model, tea.Cmd) {
	m.thinning = false
	if !m.snapshotting {
		m.loading = false
	}
	for _, d := range msg.ThinnedDates {
		m.recentThinned[d] = struct{}{}
	}
	if msg.Deleted > 0 {
		m.log.Log(logger.LevelInfo, logger.CatThinned, fmt.Sprintf(
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
		// ESTALE (stale handle) is a warning, not an error.
		level := logger.LevelError
		if msg.EstaleCount == len(msg.FailedDates) {
			level = logger.LevelWarn
		}
		m.log.Log(level, logger.CatThinned, fmt.Sprintf("Thinning: %v", msg.Err))
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
	return m, doRefresh(m.runner, m.apfsVolume, m.apfsContainer)
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
		m.snapTable.SetHeight(2)
		m.updateSnapRenderCache()
		m.clampSnapScroll()
		return
	}

	rows := make([]table.Row, 0, count)
	now := m.now()
	for i := count - 1; i >= 0; i-- {
		snap := m.snapshots[i]
		date := snap.Time.Format("2006-01-02 15:04:05")
		age := snapshot.FormatRelativeTime(snap.Time, now)

		var xid, delta, uuid, status string
		if snap.UUID != "" {
			xid = fmt.Sprintf("%d", snap.XID)
			uuid = snap.UUID

			var parts []string
			if _, pinned := m.thinPinned[snap.Date]; pinned {
				parts = append(parts, indicatorPinned+" pinned")
			}
			if snap.LimitsShrink {
				parts = append(parts, indicatorWarning+" limits shrink")
			}
			status = strings.Join(parts, " ")
			// Compute XID delta from the predecessor in ascending order.
			if i > 0 && m.snapshots[i-1].UUID != "" {
				delta = fmt.Sprintf("%d", snap.XID-m.snapshots[i-1].XID)
			}
		}
		rows = append(rows, table.Row{date, age, xid, delta, uuid, status})
	}
	m.snapTable.SetRows(rows)
	m.snapTable.SetHeight(max(len(rows)+1, 2))
	m.updateSnapRenderCache()
	m.clampSnapScroll()
}

// updateSnapAges updates only the AGE column in existing table rows.
// Called on UI ticks where only relative times change, avoiding the cost
// of recomputing column widths and rebuilding all row fields.
func (m *Model) updateSnapAges() {
	rows := m.snapTable.Rows()
	if len(rows) == 0 {
		return
	}
	now := m.now()
	count := len(m.snapshots)
	for ri, row := range rows {
		si := count - 1 - ri // newest-first mapping
		if si >= 0 && si < count {
			row[1] = snapshot.FormatRelativeTime(m.snapshots[si].Time, now)
			rows[ri] = row
		}
	}
	m.snapTable.SetRows(rows)
	m.updateSnapRenderCache()
}

func (m *Model) updateSnapRenderCache() {
	tableOut := m.snapTable.View()
	parts := strings.SplitN(tableOut, "\n", 2)
	m.snapHeaderLine = parts[0]
	if len(parts) > 1 {
		m.snapBodyLines = strings.Split(parts[1], "\n")
		return
	}
	m.snapBodyLines = nil
}

// snapTableColumns returns the column definitions for the snapshot table,
// sized to fit the current terminal width. UUID acts as the flex column,
// absorbing any remaining width so the table fills the panel.
func (m *Model) snapTableColumns() []table.Column {
	// Column.Width is the text content width. The Cell/Header styles add
	// Padding(0, 3, 0, 0) which contributes 3 extra rendered chars per column
	// (right padding only).
	const (
		colPad       = 3 // rendered padding per column (right only)
		ncols        = 6
		dateWidth    = 19 // "2006-01-02 15:04:05"
		ageWidth     = 6
		xidWidth     = 7
		deltaWidth   = 7
		uuidMinWidth = 9  // first UUID segment + ellipsis
		uuidMaxWidth = 36 // full UUID
		statusMin    = 20
	)

	tw := m.snapTable.Width()
	fixedWidth := dateWidth + ageWidth + xidWidth + deltaWidth + ncols*colPad
	remaining := tw - fixedWidth
	uuidWidth := min(max(remaining-statusMin, uuidMinWidth), uuidMaxWidth)
	statusWidth := max(remaining-uuidWidth, 0)

	return []table.Column{
		{Title: "DATE", Width: dateWidth},
		{Title: "AGE", Width: ageWidth},
		{Title: "XID", Width: xidWidth},
		{Title: "DELTA", Width: deltaWidth},
		{Title: "UUID", Width: uuidWidth},
		{Title: "STATUS", Width: statusWidth},
	}
}

// updateLogViewContent rebuilds the log viewport content.
// Entries are shown newest first so both panels scroll the same direction.
// The line at logCursor is rendered bold. Long messages wrap within the
// message column, with continuation lines indented to align.
// adjustLogCursor updates the log cursor position when the entry count or
// sequence numbers indicate new entries have arrived.
func (m *Model) adjustLogCursor(entries []logger.Entry) {
	newCount := len(entries)
	var newestSeq uint64
	if newCount > 0 {
		newestSeq = entries[newCount-1].Seq
	}

	if m.logCursor > 0 {
		if newCount > m.logCount {
			m.logCursor += newCount - m.logCount
		} else if newCount == m.logCount && newestSeq > m.logLastSeq {
			m.logCursor += int(newestSeq - m.logLastSeq)
		}
	}
	m.logLastSeq = newestSeq
	m.logCount = newCount
	if m.logCursor >= m.logCount {
		m.logCursor = max(m.logCount-1, 0)
	}
}

func (m *Model) updateLogViewContent() {
	entries := m.log.Entries()
	prevLastSeq := m.logLastSeq
	m.adjustLogCursor(entries)

	if m.logCount == 0 {
		m.logEntryY = nil
		m.logView.SetContent(m.styles.textDim.Render("(no log entries yet)"))
		return
	}

	// Prefix: "15:04:05   LEVEL CATEGORY " = 8 + 3 + 5 + 1 + 8 + 1 = 26 chars.
	const prefixW = 26
	w := m.logView.Width()
	msgW := max(w-prefixW, 10)
	indent := strings.Repeat(" ", prefixW)
	prependedLines := countPrependedLogLines(entries, prevLastSeq, msgW)

	m.logEntryY = make([]int, m.logCount)
	var b strings.Builder
	visualLine := 0
	displayIdx := 0
	for i := len(entries) - 1; i >= 0; i-- {
		if displayIdx > 0 {
			b.WriteByte('\n')
		}
		m.logEntryY[displayIdx] = visualLine
		e := entries[i]
		prefix := fmt.Sprintf("%-8s   %-5s %-8s ",
			e.Timestamp.Format("15:04:05"),
			string(e.Level),
			string(e.Category),
		)

		style := logEntryStyle(m.styles, e.Level, e.Category)
		if displayIdx == m.logCursor {
			style = style.Bold(true)
		}

		wrapped := ansi.Wordwrap(e.Message, msgW, "")
		msgLines := strings.Split(wrapped, "\n")
		for j, ml := range msgLines {
			if j > 0 {
				b.WriteByte('\n')
			}
			if j == 0 {
				b.WriteString(style.Render(prefix + ml))
			} else {
				b.WriteString(style.Render(indent + ml))
			}
		}
		visualLine += len(msgLines)
		displayIdx++
	}
	prevOffset := m.logView.YOffset()

	m.logTotalLines = visualLine
	m.logView.SetContent(b.String())

	// Preserve viewport position when the user has scrolled away from the top.
	// New entries prepend to the newest-first display, shifting everything down.
	if prevOffset > 0 && prependedLines > 0 {
		newOffset := prevOffset + prependedLines
		maxOffset := max(m.logTotalLines-m.logView.Height(), 0)
		if newOffset > maxOffset {
			newOffset = maxOffset
		}
		m.logView.SetYOffset(newOffset)
	}
}

func countPrependedLogLines(entries []logger.Entry, prevLastSeq uint64, msgW int) int {
	if prevLastSeq == 0 {
		return 0
	}

	prependedLines := 0
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Seq <= prevLastSeq {
			break
		}
		prependedLines += strings.Count(ansi.Wordwrap(entries[i].Message, msgW, ""), "\n") + 1
	}
	return prependedLines
}

// logEntryAtVisualLine returns the entry index whose visual line range
// contains the given visual line, using binary search on the sorted
// logEntryY slice. Returns -1 if out of range.
func logEntryAtVisualLine(entryY []int, totalLines, line int) int {
	if len(entryY) == 0 || line < 0 || line >= totalLines {
		return -1
	}
	// Find the last entry whose start Y <= line.
	lo, hi := 0, len(entryY)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if entryY[mid] <= line {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if entryY[lo] <= line {
		return lo
	}
	return -1
}

// logEntryStyle returns the lipgloss style for a log entry.
// Level drives the primary color; category provides secondary hints for INFO.
func logEntryStyle(s modelStyles, level logger.Level, cat logger.Category) lipgloss.Style {
	switch level {
	case logger.LevelError:
		return s.textRed
	case logger.LevelWarn:
		return s.textYellow
	default:
		switch cat {
		case logger.CatAuto:
			return s.textCyan
		case logger.CatStartup:
			return s.textMagenta
		case logger.CatCreated, logger.CatAdded:
			return s.textGreen
		case logger.CatFound:
			return s.textDim
		default:
			return s.textDefault
		}
	}
}
