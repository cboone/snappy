package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

// Model is the Bubbletea model for the Snappy TUI.
type Model struct {
	cfg    *config.Config
	runner platform.CommandRunner
	log    *logger.Logger
	auto   *snapshot.AutoManager

	snapshots     []snapshot.Snapshot
	prevSnapshots []snapshot.Snapshot
	diffAdded     int
	diffRemoved   int

	tmStatus       string
	apfsVolume     string
	otherSnapCount int
	diskInfo       string
	lastRefresh    time.Time

	width    int
	height   int
	quitting bool
	version  string

	now func() time.Time
}

// NewModel creates a Model with the given dependencies.
func NewModel(cfg *config.Config, runner platform.CommandRunner, log *logger.Logger, apfsVolume, tmStatus, version string) Model {
	now := time.Now()
	return Model{
		cfg:        cfg,
		runner:     runner,
		log:        log,
		auto:       snapshot.NewAutoManager(cfg.AutoEnabled, cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence, now),
		apfsVolume: apfsVolume,
		tmStatus:   tmStatus,
		version:    version,
		now:        time.Now,
	}
}

// Init returns the initial commands: a refresh and a tick timer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		doRefresh(m.runner, m.cfg, m.apfsVolume),
		refreshTick(m.cfg.RefreshInterval),
	)
}
