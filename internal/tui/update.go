package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/snapshot"
)

// Update handles incoming messages and returns the updated model and
// any commands to execute.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.BackgroundColorMsg:
		m.hasDarkBG = msg.IsDark()
		m.styles = newModelStyles(m.hasDarkBG)
		m.help.Styles = helpStyles(m.hasDarkBG)
		m.spinner.Style = m.styles.spinnerStyle
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

func helpStyles(hasDarkBG bool) help.Styles {
	if hasDarkBG {
		return help.DefaultDarkStyles()
	}
	return help.DefaultLightStyles()
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	cw := contentWidth(m.width)
	m.help.SetWidth(m.width)

	// Fixed-height rows: info panel + snap/log chrome + help bar.
	// Info panel: 2 borders + body lines (4 base, +1 if APFS volume present).
	// Snap & log panels: 2 borders + 1 section title each = 3 each.
	// Help bar: 1.
	infoBody := 4
	if m.apfsVolume != "" {
		infoBody++
	}
	fixedHeight := (2 + infoBody) + 3 + 3 + 1
	snapH, logH := flexPanelHeights(m.height, fixedHeight)

	m.snapView.SetWidth(cw)
	m.snapView.SetHeight(snapH)
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
		m.focusLog = !m.focusLog
		return m, nil

	case key.Matches(msg, m.keys.ScrollUp, m.keys.ScrollDown):
		return m.handleScroll(msg)
	}

	return m, nil
}

func (m Model) handleScroll(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.focusLog {
		var cmd tea.Cmd
		m.logView, cmd = m.logView.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.snapView, cmd = m.snapView.Update(msg)
	return m, cmd
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
		m.otherSnapCount = msg.APFSInfo.OtherSnapCount
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

	m.diffAdded = 0
	m.diffRemoved = 0

	if len(prev) > 0 || len(msg.Snapshots) > 0 {
		diff := snapshot.ComputeDiff(prev, msg.Snapshots)
		m.diffAdded = len(diff.Added)
		m.diffRemoved = len(diff.Removed)

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

// updateSnapViewContent rebuilds and sets the snapshot list content on the
// viewport. All snapshots are listed (newest first) for scrolling.
func (m *Model) updateSnapViewContent() {
	count := len(m.snapshots)
	if count == 0 {
		m.snapView.SetContent(m.styles.textDim.Render("(none, press 's' to create the first snapshot)"))
		return
	}

	var b strings.Builder
	for i := count - 1; i >= 0; i-- {
		if i < count-1 {
			b.WriteByte('\n')
		}
		b.WriteString(m.formatSnapshotLine(i, count))
	}
	m.snapView.SetContent(b.String())
}

// updateLogViewContent rebuilds and sets the log content on the viewport.
// Entries are shown in chronological order (oldest first); the viewport
// auto-scrolls to the bottom so the newest entry is always visible.
func (m *Model) updateLogViewContent() {
	entries := m.log.Entries()
	if len(entries) == 0 {
		m.logView.SetContent(m.styles.textDim.Render("(no log entries yet)"))
		return
	}

	var b strings.Builder
	for i, entry := range entries {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(m.colorizeLogEntry(entry))
	}
	m.logView.SetContent(b.String())
	m.logView.GotoBottom()
}
