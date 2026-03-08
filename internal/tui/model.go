package tui

import (
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

type keyMap struct {
	Snapshot   key.Binding
	Refresh    key.Binding
	AutoSnap   key.Binding
	OpenLog    key.Binding
	Quit       key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Tab        key.Binding
	ShiftTab   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Snapshot, k.Refresh, k.AutoSnap, k.OpenLog, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Snapshot, k.Refresh, k.AutoSnap, k.OpenLog, k.Quit},
		{k.ScrollUp, k.ScrollDown, k.Tab, k.ShiftTab},
	}
}

func defaultKeyMap() keyMap {
	return keyMap{
		Snapshot: key.NewBinding(
			key.WithKeys("s", "S"),
			key.WithHelp("s", "snapshot"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "R"),
			key.WithHelp("r", "refresh"),
		),
		AutoSnap: key.NewBinding(
			key.WithKeys("a", "A"),
			key.WithHelp("a", "auto-snap"),
		),
		OpenLog: key.NewBinding(
			key.WithKeys("l", "L"),
			key.WithHelp("l", "open log"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "Q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "scroll down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
	}
}

// Panel focus constants.
const (
	panelInfo = iota
	panelSnap
	panelLog
)

// Model is the Bubbletea model for the Snappy TUI.
type Model struct {
	cfg    *config.Config
	runner platform.CommandRunner
	log    *logger.Logger
	auto   *snapshot.AutoManager

	snapshots     []snapshot.Snapshot
	prevSnapshots []snapshot.Snapshot

	tmStatus           string
	apfsVolume         string
	apfsContainer      string
	volumeName         string
	lastOtherSnapCount int
	lastTidemarkErr    string
	diskInfo           string
	tidemark           string
	lastRefresh        time.Time
	daemonActive       bool

	width              int
	height             int
	quitting           bool
	refreshing         bool
	refreshPending     bool
	snapshotting       bool
	autoSnapshotting   bool
	thinning           bool
	thinPinned         map[string]struct{}
	recentCreated      map[string]struct{}
	recentThinned      map[string]struct{}
	hadFirstRefresh    bool
	lastRefreshSummary string
	version            string

	keys          keyMap
	help          help.Model
	snapTable     table.Model
	logView       viewport.Model
	logCursor     int
	logCount      int
	logLastSeq    uint64
	logEntryY     []int
	logTotalLines int
	spinner       spinner.Model
	styles        modelStyles
	loading       bool
	focusPanel    int
	flash         flashState
	hasDarkBG     bool

	snapPanelY       int
	logPanelY        int
	helpBarY         int
	snapScrollOffset int
	snapVisibleRows  int
	snapHeaderLine   string
	snapBodyLines    []string

	now func() time.Time
}

// ModelParams groups the string and boolean parameters for NewModel,
// preventing accidental argument reordering at call sites.
type ModelParams struct {
	APFSVolume    string
	APFSContainer string
	TMStatus      string
	VolumeName    string
	Version       string
	DaemonActive  bool
}

// NewModel creates a Model with the given dependencies. When DaemonActive is
// true, auto-snapshots are disabled because a background service holds the lock.
func NewModel(cfg *config.Config, runner platform.CommandRunner, log *logger.Logger, params ModelParams) Model {
	now := time.Now()
	hasDarkBG := true

	autoEnabled := cfg.AutoEnabled
	if params.DaemonActive {
		autoEnabled = false
	}

	keys := defaultKeyMap()
	styles := newModelStyles(hasDarkBG)

	h := help.New()
	h.SetWidth(80)
	h.Styles = helpStyles(styles)

	const defaultTableHeight = 10
	st := table.New(
		table.WithWidth(76),
		table.WithHeight(defaultTableHeight),
		table.WithFocused(true),
		table.WithStyles(styles.tableStyles),
	)

	lv := viewport.New(viewport.WithWidth(76), viewport.WithHeight(10))

	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(styles.spinnerStyle),
	)

	m := Model{
		cfg:             cfg,
		runner:          runner,
		log:             log,
		auto:            snapshot.NewAutoManager(autoEnabled, cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence, now),
		apfsVolume:      params.APFSVolume,
		tmStatus:        params.TMStatus,
		volumeName:      params.VolumeName,
		daemonActive:    params.DaemonActive,
		apfsContainer:   params.APFSContainer,
		refreshing:      true,
		thinPinned:      make(map[string]struct{}),
		recentCreated:   make(map[string]struct{}),
		recentThinned:   make(map[string]struct{}),
		version:         params.Version,
		width:           80,
		height:          24,
		keys:            keys,
		help:            h,
		snapTable:       st,
		snapVisibleRows: defaultTableHeight - 1, // minus header row
		logView:         lv,
		spinner:         s,
		styles:          styles,
		focusPanel:      panelSnap,
		hasDarkBG:       hasDarkBG,
		now:             time.Now,
	}

	m.updateSnapViewContent()
	m.updateLogViewContent()

	return m
}

// Init returns the initial commands: a refresh, a tick timer, and a
// background color request. The UI tick is only started when
// auto-snapshot is enabled, since it drives the countdown timer.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		doRefresh(m.runner, m.apfsVolume, m.apfsContainer),
		refreshTick(m.cfg.RefreshInterval),
		tea.RequestBackgroundColor,
	}
	if m.auto.Enabled() {
		cmds = append(cmds, uiTick())
	}
	return tea.Batch(cmds...)
}
