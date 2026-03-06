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
	Quit       key.Binding
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Tab        key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Snapshot, k.Refresh, k.AutoSnap, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Snapshot, k.Refresh, k.AutoSnap, k.Quit},
		{k.ScrollUp, k.ScrollDown, k.Tab},
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
			key.WithHelp("tab", "focus"),
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
	volumeName         string
	lastOtherSnapCount int
	diskInfo           string
	lastRefresh        time.Time
	daemonActive       bool

	width          int
	height         int
	quitting       bool
	refreshing     bool
	refreshPending bool
	snapshotting   bool
	thinning       bool
	thinPinned     map[string]struct{}
	version        string

	keys          keyMap
	help          help.Model
	snapTable     table.Model
	logView       viewport.Model
	logCursor     int
	logCount      int
	logEntryY     []int
	logTotalLines int
	spinner       spinner.Model
	styles        modelStyles
	loading       bool
	focusPanel    int
	hasDarkBG     bool

	snapPanelY int
	logPanelY  int
	helpBarY   int

	now func() time.Time
}

// NewModel creates a Model with the given dependencies. When daemonActive is
// true, auto-snapshots are disabled because a background service holds the lock.
func NewModel(cfg *config.Config, runner platform.CommandRunner, log *logger.Logger, apfsVolume, tmStatus, volumeName, version string, daemonActive bool) Model {
	now := time.Now()
	hasDarkBG := true

	autoEnabled := cfg.AutoEnabled
	if daemonActive {
		autoEnabled = false
	}

	keys := defaultKeyMap()
	styles := newModelStyles(hasDarkBG)

	h := help.New()
	h.SetWidth(80)
	h.Styles = helpStyles(styles)

	st := table.New(
		table.WithWidth(76),
		table.WithHeight(10),
		table.WithFocused(true),
		table.WithStyles(styles.tableStyles),
	)

	lv := viewport.New(viewport.WithWidth(76), viewport.WithHeight(10))

	s := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(styles.spinnerStyle),
	)

	m := Model{
		cfg:          cfg,
		runner:       runner,
		log:          log,
		auto:         snapshot.NewAutoManager(autoEnabled, cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence, now),
		apfsVolume:   apfsVolume,
		tmStatus:     tmStatus,
		volumeName:   volumeName,
		daemonActive: daemonActive,
		refreshing:   true,
		thinPinned:   make(map[string]struct{}),
		version:      version,
		width:        80,
		height:       24,
		keys:         keys,
		help:         h,
		snapTable:    st,
		logView:      lv,
		spinner:      s,
		styles:       styles,
		focusPanel:   panelSnap,
		hasDarkBG:    hasDarkBG,
		now:          time.Now,
	}

	m.updateSnapViewContent()
	m.updateLogViewContent()

	return m
}

// Init returns the initial commands: a refresh, a tick timer, and a
// background color request.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		doRefresh(m.runner, m.cfg, m.apfsVolume),
		refreshTick(m.cfg.RefreshInterval),
		tea.RequestBackgroundColor,
		uiTick(),
	)
}
