package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/snapshot"
)

// Update handles incoming messages and returns the updated model and
// any commands to execute.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S":
		m.log.Log(logger.Info, "Creating snapshot...")
		return m, doCreateSnapshot(m.runner)

	case "r", "R":
		return m, doRefresh(m.runner, m.cfg, m.apfsVolume)

	case "a", "A":
		now := m.now()
		enabled := m.auto.Toggle(now)
		if enabled {
			m.log.Log(logger.Info, fmt.Sprintf(
				"Auto-snapshots enabled (every %ds, thin >%ds to %ds)",
				int(m.auto.Interval().Seconds()),
				int(m.auto.ThinAge().Seconds()),
				int(m.auto.ThinCadence().Seconds()),
			))
		} else {
			m.log.Log(logger.Info, "Auto-snapshots disabled")
		}
		return m, nil

	case "q", "Q", "ctrl+c":
		m.log.Log(logger.Info, "Shutting down")
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleTick() (tea.Model, tea.Cmd) {
	now := m.now()

	var cmds []tea.Cmd

	if m.auto.ShouldSnapshot(now) {
		m.auto.RecordSnapshot(now)
		m.log.Log(logger.Auto, "Creating auto-snapshot...")
		cmds = append(cmds, doCreateSnapshot(m.runner))
	}

	cmds = append(cmds, doRefresh(m.runner, m.cfg, m.apfsVolume))
	cmds = append(cmds, refreshTick(m.cfg.RefreshInterval))

	return m, tea.Batch(cmds...)
}

func (m Model) handleRefreshResult(msg RefreshResultMsg) (tea.Model, tea.Cmd) {
	m.tmStatus = msg.TMStatus

	if msg.APFSInfo.Volume != "" {
		m.apfsVolume = msg.APFSInfo.Volume
		m.otherSnapCount = msg.APFSInfo.OtherSnapCount
	}

	if !msg.DiskErr {
		m.diskInfo = msg.DiskInfo.String()
	}

	m.lastRefresh = m.now()

	// Compute diff
	prev := m.snapshots
	m.prevSnapshots = prev
	m.snapshots = msg.Snapshots

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

	// Check thinning
	var cmds []tea.Cmd
	targets := m.auto.ComputeThinTargets(m.snapshots, m.now())
	if len(targets) > 0 {
		cmds = append(cmds, doThinSnapshots(m.runner, targets))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleSnapshotCreated(msg SnapshotCreatedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.log.Log(logger.Error, fmt.Sprintf("Failed to create snapshot: %v", msg.Err))
	} else if msg.Date != "" {
		m.log.Log(logger.Created, "Snapshot created: "+msg.Date)
	} else {
		m.log.Log(logger.Created, "Snapshot created")
	}
	return m, doRefresh(m.runner, m.cfg, m.apfsVolume)
}

func (m Model) handleThinResult(msg ThinResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.log.Log(logger.Error, fmt.Sprintf("Thinning error: %v", msg.Err))
	} else if msg.Deleted > 0 {
		m.log.Log(logger.Thinned, fmt.Sprintf(
			"Thinned %d snapshot(s) older than %dm to %ds cadence",
			msg.Deleted,
			int(m.auto.ThinAge().Minutes()),
			int(m.auto.ThinCadence().Seconds()),
		))
	}
	return m, doRefresh(m.runner, m.cfg, m.apfsVolume)
}
