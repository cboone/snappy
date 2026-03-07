package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/service"
	"github.com/cboone/snappy/internal/snapshot"
)

type mockRunner struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	output []byte
	err    error
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}
	return nil, fmt.Errorf("unmocked command: %s", key)
}

func testConfig() *config.Config {
	return &config.Config{
		RefreshInterval:      60 * time.Second,
		LogDir:               "",
		AutoEnabled:          true,
		AutoSnapshotInterval: 60 * time.Second,
		ThinAgeThreshold:     600 * time.Second,
		ThinCadence:          300 * time.Second,
	}
}

func testModel() Model {
	cfg := testConfig()
	log := logger.New(logger.Options{MaxEntries: 50})
	runner := &mockRunner{responses: map[string]mockResponse{}}
	m := NewModel(cfg, runner, log, ModelParams{
		APFSVolume:    "disk3s5",
		APFSContainer: "disk3",
		TMStatus:      "Configured",
		VolumeName:    "/",
		Version:       "dev",
	})
	m.width = 80
	m.height = 40
	return m
}

// viewContent extracts the rendered string from a tea.View.
func viewContent(m Model) string {
	return m.View().Content
}

func TestViewEmpty(t *testing.T) {
	m := testModel()
	v := viewContent(m)

	if !strings.Contains(v, "snappy") {
		t.Error("view missing snappy title")
	}
	if !strings.Contains(v, "local snapshots (0)") {
		t.Error("view missing snapshot count")
	}
	if !strings.Contains(v, "press 's'") {
		t.Error("view missing empty state hint")
	}
	if !strings.Contains(v, "snapshot") {
		t.Error("view missing controls")
	}
}

func TestViewWithSnapshots(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.snapshots = []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute), UUID: "E3F52B7A-8C19-4D6E-A031-7F5B2E9D14C8", XID: 1547201, Purgeable: true},
		{Date: "2026-03-01-144000", Time: now.Add(-20 * time.Minute), UUID: "9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5", XID: 1547289, Purgeable: true},
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute), UUID: "B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04", XID: 1547356, Purgeable: true},
	}
	m.log.Log(logger.LevelInfo, logger.CatStartup, "version=dev apfs-volume=disk3s5")
	m.log.Log(logger.LevelInfo, logger.CatRefresh, "3 snapshots, 0 other APFS snapshots")
	m.log.Log(logger.LevelInfo, logger.CatRefresh, "disk: 460Gi total, 215Gi used (48%)")
	m.updateSnapViewContent()
	m.updateLogViewContent()

	v := viewContent(m)

	if !strings.Contains(v, "local snapshots (3)") {
		t.Error("view missing correct snapshot count")
	}
	if !strings.Contains(v, "2026-03-01 14:50:00") {
		t.Error("view missing newest snapshot")
	}
	if !strings.Contains(v, "2026-03-01 14:30:00") {
		t.Error("view missing oldest snapshot")
	}
}

func TestViewAllSnapshotsInViewport(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	uuids := []string{
		"E3F52B7A-8C19-4D6E-A031-7F5B2E9D14C8",
		"9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5",
		"B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04",
		"4F6D1A2E-7C8B-4935-BE12-9A3D5F7E6C81",
		"D2A8F3C7-1E4B-4D69-8F05-6B3C9A7E2D14",
		"71E5B9A3-3F82-4C17-A6D4-8E2F1B5C93A0",
	}
	var snaps []snapshot.Snapshot
	for i := range 6 {
		d := now.Add(-time.Duration(60-i*10) * time.Minute)
		snaps = append(snaps, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      uuids[i],
			XID:       1547200 + i*73,
			Purgeable: true,
		})
	}
	m.snapshots = snaps
	m.updateSnapViewContent()

	v := viewContent(m)

	if !strings.Contains(v, "local snapshots (6)") {
		t.Error("view missing correct snapshot count")
	}
	// All 6 snapshots should be in the viewport content (no bookend/ellipsis).
	for i, snap := range snaps {
		formatted := snap.Time.Format("2006-01-02 15:04:05")
		if !strings.Contains(v, formatted) {
			t.Errorf("snapshot %d (%s) missing from viewport", i, formatted)
		}
	}
}

func TestViewQuitting(t *testing.T) {
	m := testModel()
	m.quitting = true
	v := m.View()
	if v.AltScreen {
		t.Error("View() when quitting should not use alt screen")
	}
	if v.Content == "" {
		t.Error("View() when quitting should render final frame")
	}
}

func TestKeyQuit(t *testing.T) {
	m := testModel()
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	model := updated.(Model)
	if !model.quitting {
		t.Error("expected quitting = true after 'q'")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

func TestKeyToggleAuto(t *testing.T) {
	m := testModel()

	// Initially enabled, toggle off
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := updated.(Model)
	if model.auto.Enabled() {
		t.Error("expected auto disabled after toggle")
	}

	// Toggle back on
	updated, _ = model.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model = updated.(Model)
	if !model.auto.Enabled() {
		t.Error("expected auto enabled after second toggle")
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	model := updated.(Model)
	if model.width != 120 || model.height != 50 {
		t.Errorf("size = %dx%d, want 120x50", model.width, model.height)
	}
}

func TestRefreshResultMsg(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute), UUID: "E3F52B7A-8C19-4D6E-A031-7F5B2E9D14C8", XID: 1547201, Purgeable: true},
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute), UUID: "B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04", XID: 1547356, Purgeable: true},
	}

	msg := RefreshResultMsg{
		Snapshots: snaps,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	if len(model.snapshots) != 2 {
		t.Errorf("snapshots = %d, want 2", len(model.snapshots))
	}
	if model.tmStatus != "Configured" {
		t.Errorf("tmStatus = %q, want Configured", model.tmStatus)
	}
	if !strings.Contains(model.diskInfo, "460Gi") {
		t.Errorf("diskInfo = %q, want to contain 460Gi", model.diskInfo)
	}
}

func TestRefreshResultMsgSnapshotErrorKeepsExistingSnapshots(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.snapshots = []snapshot.Snapshot{
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute), UUID: "B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04", XID: 1547356, Purgeable: true},
	}

	updated, _ := m.Update(RefreshResultMsg{SnapshotErr: fmt.Errorf("tmutil failed")})
	model := updated.(Model)

	if len(model.snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(model.snapshots))
	}

	entries := model.log.Entries()
	var removedCount int
	var sawError bool
	for _, e := range entries {
		if e.Category == logger.CatRemoved {
			removedCount++
		}
		if e.Level == logger.LevelError && strings.Contains(e.Message, "Failed to list snapshots") {
			sawError = true
		}
	}

	if removedCount != 0 {
		t.Fatalf("got %d removal log entries, want 0", removedCount)
	}
	if !sawError {
		t.Fatal("expected refresh error log entry")
	}
}

func TestSnapshotCreatedMsg(t *testing.T) {
	m := testModel()
	m.refreshing = false

	updated, cmd := m.Update(SnapshotCreatedMsg{Date: "2026-03-01-150000"})
	_ = updated.(Model)
	if cmd == nil {
		t.Error("expected refresh command after snapshot created")
	}

	entries := m.log.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Message, "2026-03-01-150000") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log entry for created snapshot")
	}
}

func TestSnapshotCreatedDuringRefreshSetsRefreshPending(t *testing.T) {
	m := testModel()
	m.refreshing = true

	updated, cmd := m.Update(SnapshotCreatedMsg{Date: "2026-03-01-150000"})
	model := updated.(Model)
	if cmd != nil {
		t.Error("expected nil command when refresh already in flight")
	}
	if !model.refreshPending {
		t.Error("expected refreshPending = true")
	}
}

func TestRefreshResultTriggersFollowUpWhenPending(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.refreshPending = true

	updated, cmd := m.Update(RefreshResultMsg{
		Snapshots: []snapshot.Snapshot{},
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	if model.refreshPending {
		t.Error("expected refreshPending = false after processing")
	}
	if !model.refreshing {
		t.Error("expected refreshing = true for follow-up refresh")
	}
	if cmd == nil {
		t.Error("expected follow-up refresh command")
	}
}

func TestSnapshotCreatedMsgError(t *testing.T) {
	m := testModel()

	updated, _ := m.Update(SnapshotCreatedMsg{Err: fmt.Errorf("permission denied")})
	_ = updated.(Model)

	entries := m.log.Entries()
	found := false
	for _, e := range entries {
		if e.Level == logger.LevelError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error log entry")
	}
}

func TestViewAutoStatusOn(t *testing.T) {
	m := testModel()
	v := viewContent(m)
	if !strings.Contains(v, "on") {
		t.Error("view should show auto-snapshot 'on'")
	}
}

func TestViewAutoStatusOff(t *testing.T) {
	m := testModel()
	m.auto.Toggle(m.now())
	v := viewContent(m)
	if !strings.Contains(v, "off") {
		t.Error("view should show auto-snapshot 'off'")
	}
}

func TestViewAutoStatusDaemon(t *testing.T) {
	cfg := testConfig()
	log := logger.New(logger.Options{MaxEntries: 50})
	runner := &mockRunner{responses: map[string]mockResponse{}}
	m := NewModel(cfg, runner, log, ModelParams{
		APFSVolume:    "disk3s5",
		APFSContainer: "disk3",
		TMStatus:      "Configured",
		VolumeName:    "/",
		Version:       "dev",
		DaemonActive:  true,
	})
	m.width = 80
	m.height = 40

	v := viewContent(m)
	if !strings.Contains(v, "service") {
		t.Error("view should show 'service' when background service is active")
	}
	if m.auto.Enabled() {
		t.Error("auto-snapshots should be disabled when daemon is active")
	}
}

func TestAutoToggleIgnoredWhenDaemonActive(t *testing.T) {
	cfg := testConfig()
	log := logger.New(logger.Options{MaxEntries: 50})
	runner := &mockRunner{responses: map[string]mockResponse{}}
	m := NewModel(cfg, runner, log, ModelParams{
		APFSVolume:    "disk3s5",
		APFSContainer: "disk3",
		TMStatus:      "Configured",
		VolumeName:    "/",
		Version:       "dev",
		DaemonActive:  true,
	})
	m.width = 80
	m.height = 40

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := updated.(Model)

	if model.auto.Enabled() {
		t.Error("auto-snapshots should remain disabled when daemon is active")
	}

	entries := model.log.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Message, "background service") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log message about daemon managing auto-snapshots")
	}
}

func TestRefreshTickDisablesAutoWhenDaemonLockAppears(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.cfg.LogDir = t.TempDir()
	m.auto.RecordSnapshot(now.Add(-2 * m.auto.Interval()))

	lockPath := service.DefaultLockPath(m.cfg.LogDir)
	lock, err := service.Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer func() { _ = lock.Release() }()

	updated, _ := m.Update(RefreshTickMsg{})
	model := updated.(Model)

	if !model.daemonActive {
		t.Error("expected daemonActive = true after lock appears")
	}
	if model.auto.Enabled() {
		t.Error("expected auto-snapshots disabled after lock appears")
	}
	if model.snapshotting {
		t.Error("expected no auto-snapshot while daemon lock is held")
	}

	entries := model.log.Entries()
	found := false
	for _, e := range entries {
		if strings.Contains(e.Message, "Background service detected") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected lock-detection log message")
	}
}

func TestRefreshTickClearsDaemonActiveWhenLockReleased(t *testing.T) {
	cfg := testConfig()
	cfg.LogDir = t.TempDir()
	log := logger.New(logger.Options{MaxEntries: 50})
	runner := &mockRunner{responses: map[string]mockResponse{}}
	m := NewModel(cfg, runner, log, ModelParams{
		APFSVolume:    "disk3s5",
		APFSContainer: "disk3",
		TMStatus:      "Configured",
		VolumeName:    "/",
		Version:       "dev",
		DaemonActive:  true,
	})
	m.now = func() time.Time {
		return time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	}

	updated, _ := m.Update(RefreshTickMsg{})
	model := updated.(Model)

	if model.daemonActive {
		t.Error("expected daemonActive = false when lock is no longer held")
	}

	updated, _ = model.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model = updated.(Model)
	if !model.auto.Enabled() {
		t.Error("expected auto-snapshots to be toggleable after daemon lock release")
	}
}

func TestViewAPFSDetails(t *testing.T) {
	m := testModel()
	m.width = 120
	m.snapTable.SetWidth(contentWidth(120))
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.snapshots = []snapshot.Snapshot{
		{
			Date:         "2026-03-01-145000",
			Time:         now.Add(-10 * time.Minute),
			UUID:         "B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04",
			XID:          1547356,
			Purgeable:    true,
			LimitsShrink: false,
		},
		{
			Date:         "2026-03-01-144000",
			Time:         now.Add(-20 * time.Minute),
			UUID:         "9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5",
			XID:          1547289,
			Purgeable:    true,
			LimitsShrink: true,
		},
	}
	m.updateSnapViewContent()

	v := viewContent(m)

	if !strings.Contains(v, "B7C83E91") {
		t.Error("view missing UUID for first snapshot")
	}
	if !strings.Contains(v, "limits shrink") {
		t.Error("view missing limits-shrink flag for second snapshot")
	}
}

func TestExactlyFourSnapshots(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	uuids := []string{
		"E3F52B7A-8C19-4D6E-A031-7F5B2E9D14C8",
		"9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5",
		"B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04",
		"4F6D1A2E-7C8B-4935-BE12-9A3D5F7E6C81",
	}
	for i := range 4 {
		d := now.Add(-time.Duration(40-i*10) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      uuids[i],
			XID:       1547200 + i*73,
			Purgeable: true,
		})
	}
	m.updateSnapViewContent()

	v := viewContent(m)
	if strings.Contains(v, "... and") {
		t.Error("exactly 4 snapshots should not show ellipsis")
	}
}

func TestSnapshotKeyIgnoredWhileSnapshotting(t *testing.T) {
	m := testModel()
	m.snapshotting = true

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model := updated.(Model)

	if cmd != nil {
		t.Error("expected nil command when snapshot already in flight")
	}
	if !model.snapshotting {
		t.Error("expected snapshotting to remain true")
	}
}

func TestSnapshotCreatedClearsSnapshottingFlag(t *testing.T) {
	m := testModel()
	m.snapshotting = true
	m.refreshing = false

	updated, _ := m.Update(SnapshotCreatedMsg{Date: "2026-03-01-150000"})
	model := updated.(Model)

	if model.snapshotting {
		t.Error("expected snapshotting = false after SnapshotCreatedMsg")
	}
}

func TestThinResultClearsThinningFlag(t *testing.T) {
	m := testModel()
	m.thinning = true
	m.loading = true
	m.refreshing = false

	updated, _ := m.Update(ThinResultMsg{Deleted: 1})
	model := updated.(Model)

	if model.thinning {
		t.Error("expected thinning = false after ThinResultMsg")
	}
	if model.loading {
		t.Error("expected loading = false after ThinResultMsg")
	}
}

func TestDoThinSnapshotsReportsDeleteFailures(t *testing.T) {
	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil deletelocalsnapshots 2026-03-01-140000": {output: []byte("Deleted\n")},
		"tmutil deletelocalsnapshots 2026-03-01-140100": {err: fmt.Errorf("permission denied")},
	}}

	msg := doThinSnapshots(runner, []string{"2026-03-01-140000", "2026-03-01-140100"})()
	result, ok := msg.(ThinResultMsg)
	if !ok {
		t.Fatalf("msg type = %T, want ThinResultMsg", msg)
	}

	if result.Deleted != 1 {
		t.Fatalf("Deleted = %d, want 1", result.Deleted)
	}
	if result.Err == nil {
		t.Fatal("expected non-nil error for failed deletions")
	}
	if len(result.FailedDates) != 1 || result.FailedDates[0] != "2026-03-01-140100" {
		t.Fatalf("FailedDates = %v, want [2026-03-01-140100]", result.FailedDates)
	}
}

func TestViewSpinnerDuringLoading(t *testing.T) {
	m := testModel()

	// Without loading, the border title should not contain an operation label.
	noLoading := viewContent(m)
	if strings.Contains(noLoading, "Refreshing") {
		t.Error("expected no Refreshing label when not loading")
	}

	m.loading = true
	withLoading := viewContent(m)

	if !strings.Contains(withLoading, "snappy") {
		t.Error("view missing snappy title during loading")
	}
	// The border title should include the operation label when loading.
	if !strings.Contains(withLoading, "Refreshing") {
		t.Error("expected Refreshing label in border title when loading")
	}
}

func TestViewFullHeight(t *testing.T) {
	m := testModel()
	m.width = 100
	m.height = 40

	// Trigger resize to recalculate viewport dimensions
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model := updated.(Model)

	v := model.View().Content
	lines := strings.Split(v, "\n")
	// The output should have a reasonable number of lines approaching terminal height
	if len(lines) < 20 {
		t.Errorf("expected at least 20 lines for full-height TUI, got %d", len(lines))
	}
}

func TestLogViewShowsNewestFirst(t *testing.T) {
	m := testModel()
	m.logView.SetHeight(3)

	for i := range 8 {
		m.log.Log(logger.LevelInfo, logger.CatRefresh, fmt.Sprintf("entry-%d", i))
	}
	m.updateLogViewContent()

	// Newest entry should be at the top (cursor=0).
	v := m.logView.View()
	if !strings.Contains(v, "entry-7") {
		t.Fatal("expected newest log entry to be visible at top")
	}
}

func TestMouseClickSnapshotSelectsTopVisibleRowWhenScrolled(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	for i := range 8 {
		d := now.Add(-time.Duration(8-i) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			XID:       1547200 + i,
			Purgeable: true,
		})
	}
	m.updateSnapViewContent()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	model := updated.(Model)

	// Scroll viewport down without moving cursor via mouse wheel.
	updated, _ = model.Update(tea.MouseWheelMsg{
		Button: tea.MouseWheelDown,
		Y:      model.snapPanelY + 2,
	})
	model = updated.(Model)

	if got := model.snapScrollOffset; got != 1 {
		t.Fatalf("snapScrollOffset = %d, want 1 after wheel down", got)
	}
	if got := model.snapTable.Cursor(); got != 0 {
		t.Fatalf("cursor after wheel = %d, want 0 (unchanged)", got)
	}

	// Click the top visible line: should select row at snapScrollOffset.
	updated, _ = model.Update(tea.MouseClickMsg{
		Button: tea.MouseLeft,
		Y:      model.snapPanelY + 2,
	})
	model = updated.(Model)

	if got := model.snapTable.Cursor(); got != 1 {
		t.Fatalf("cursor after click = %d, want 1 (top visible row)", got)
	}
}

func TestMouseWheelSnapScrollsWithoutChangingCursor(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	for i := range 10 {
		d := now.Add(-time.Duration(10-i) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			XID:       1547200 + i,
			Purgeable: true,
		})
	}
	m.updateSnapViewContent()

	// Height 16: snapH=3, snapVisibleRows=2, so 10 rows have maxOffset=8.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	model := updated.(Model)

	cursorBefore := model.snapTable.Cursor()

	// Scroll down 3 times via mouse wheel.
	for range 3 {
		updated, _ = model.Update(tea.MouseWheelMsg{
			Button: tea.MouseWheelDown,
			Y:      model.snapPanelY + 2,
		})
		model = updated.(Model)
	}

	if got := model.snapScrollOffset; got != 3 {
		t.Errorf("snapScrollOffset = %d, want 3 after 3 wheel-downs", got)
	}
	if got := model.snapTable.Cursor(); got != cursorBefore {
		t.Errorf("cursor = %d, want %d (unchanged by wheel)", got, cursorBefore)
	}

	// Scroll back up.
	updated, _ = model.Update(tea.MouseWheelMsg{
		Button: tea.MouseWheelUp,
		Y:      model.snapPanelY + 2,
	})
	model = updated.(Model)

	if got := model.snapScrollOffset; got != 2 {
		t.Errorf("snapScrollOffset = %d, want 2 after wheel-up", got)
	}
}

func TestMouseWheelLogScrollsWithoutChangingCursor(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := updated.(Model)

	// Add enough log entries to overflow the viewport.
	for i := range 20 {
		model.log.Log(logger.LevelInfo, logger.CatRefresh, fmt.Sprintf("entry-%d", i))
	}
	model.updateLogViewContent()

	cursorBefore := model.logCursor

	// Scroll down via mouse wheel inside the log panel.
	updated, _ = model.Update(tea.MouseWheelMsg{
		Button: tea.MouseWheelDown,
		Y:      model.logPanelY + 1,
	})
	model = updated.(Model)

	if got := model.logCursor; got != cursorBefore {
		t.Errorf("logCursor = %d, want %d (unchanged by wheel)", got, cursorBefore)
	}
	if got := model.logView.YOffset(); got != 1 {
		t.Errorf("logView.YOffset = %d, want 1 after wheel-down", got)
	}
}

func TestKeyboardScrollSnapMovesCursorAndAutoScrolls(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	for i := range 10 {
		d := now.Add(-time.Duration(10-i) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			XID:       1547200 + i,
			Purgeable: true,
		})
	}
	m.updateSnapViewContent()

	// Height 16: snapH=3, snapVisibleRows=2.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	model := updated.(Model)

	if model.snapVisibleRows != 2 {
		t.Fatalf("snapVisibleRows = %d, want 2", model.snapVisibleRows)
	}

	// Move cursor down past visible rows: should auto-scroll.
	for range 3 {
		updated, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
		model = updated.(Model)
	}

	if got := model.snapTable.Cursor(); got != 3 {
		t.Errorf("cursor = %d, want 3 after 3 j presses", got)
	}
	// Cursor should be visible: snapScrollOffset <= 3 < snapScrollOffset + 2.
	if model.snapTable.Cursor() < model.snapScrollOffset ||
		model.snapTable.Cursor() >= model.snapScrollOffset+model.snapVisibleRows {
		t.Errorf("cursor %d not visible with offset=%d visibleRows=%d",
			model.snapTable.Cursor(), model.snapScrollOffset, model.snapVisibleRows)
	}
}

func TestSnapScrollOffsetClampedOnRowDecrease(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	for i := range 10 {
		d := now.Add(-time.Duration(10-i) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			XID:       1547200 + i,
			Purgeable: true,
		})
	}
	m.updateSnapViewContent()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := updated.(Model)

	// Scroll near the bottom.
	model.snapScrollOffset = 8
	model.clampSnapScroll()

	// Reduce row count: offset should be clamped.
	model.snapshots = model.snapshots[:3]
	model.updateSnapViewContent()

	maxOffset := max(len(model.snapTable.Rows())-model.snapVisibleRows, 0)
	if model.snapScrollOffset > maxOffset {
		t.Errorf("snapScrollOffset = %d, want <= %d after row decrease",
			model.snapScrollOffset, maxOffset)
	}
}

func TestEnsureSnapCursorVisible(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	for i := range 10 {
		d := now.Add(-time.Duration(10-i) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			XID:       1547200 + i,
			Purgeable: true,
		})
	}
	m.updateSnapViewContent()

	// Height 16: snapH=3, snapVisibleRows=2.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 16})
	model := updated.(Model)

	// Cursor below visible range: should scroll down.
	model.snapTable.SetCursor(5)
	model.ensureSnapCursorVisible()
	if model.snapScrollOffset > 5 || model.snapScrollOffset+model.snapVisibleRows <= 5 {
		t.Errorf("cursor 5 not visible: offset=%d visibleRows=%d",
			model.snapScrollOffset, model.snapVisibleRows)
	}

	// Cursor above visible range: should scroll up.
	model.snapScrollOffset = 5
	model.snapTable.SetCursor(2)
	model.ensureSnapCursorVisible()
	if model.snapScrollOffset > 2 {
		t.Errorf("cursor 2 not visible: offset=%d", model.snapScrollOffset)
	}
}

func TestRenderSnapshotPanelClipsWithOffset(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	for i := range 10 {
		d := now.Add(-time.Duration(10-i) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date:      d.Format("2006-01-02-150405"),
			Time:      d,
			UUID:      fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			XID:       1547200 + i,
			Purgeable: true,
		})
	}
	m.updateSnapViewContent()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model := updated.(Model)

	// Render at offset 0.
	panel0 := model.renderSnapshotPanel(model.width)

	// Scroll down.
	model.snapScrollOffset = 3
	model.clampSnapScroll()
	panel3 := model.renderSnapshotPanel(model.width)

	// Both panels should have the same number of lines (consistent height).
	lines0 := strings.Count(panel0, "\n")
	lines3 := strings.Count(panel3, "\n")
	if lines0 != lines3 {
		t.Errorf("panel line count offset=0: %d, offset=3: %d, want equal", lines0, lines3)
	}

	// Content should differ (different rows visible).
	if panel0 == panel3 {
		t.Error("panel content identical at offset 0 and 3, want different visible rows")
	}
}

func TestSnapshotPanelKeepsViewportHeightWhenEmpty(t *testing.T) {
	m := testModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model := updated.(Model)

	emptyLines := strings.Count(model.renderSnapshotPanel(model.width), "\n")

	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	model.now = func() time.Time { return now }
	model.snapshots = []snapshot.Snapshot{{
		Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute),
		UUID: "B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04", XID: 1547356, Purgeable: true,
	}}
	model.updateSnapViewContent()

	nonEmptyLines := strings.Count(model.renderSnapshotPanel(model.width), "\n")
	if emptyLines != nonEmptyLines {
		t.Fatalf("snapshot panel lines empty=%d non-empty=%d, want equal fixed-height layout", emptyLines, nonEmptyLines)
	}
}

func TestRefreshResultStartsSpinnerWhenThinning(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	m.snapshots = nil
	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-130000", Time: now.Add(-2 * time.Hour), UUID: "E3F52B7A-8C19-4D6E-A031-7F5B2E9D14C8", XID: 1547201, Purgeable: true},
		{Date: "2026-03-01-130100", Time: now.Add(-119 * time.Minute), UUID: "9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5", XID: 1547289, Purgeable: true},
	}

	updated, cmd := m.Update(RefreshResultMsg{
		Snapshots: snaps,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	if cmd == nil {
		t.Fatal("expected command batch to include thinning and spinner")
	}
	if !model.thinning {
		t.Fatal("expected thinning=true when thin targets are found")
	}
	if !model.loading {
		t.Fatal("expected loading=true while thinning is in progress")
	}
}

func TestThinResultErrorNoRefresh(t *testing.T) {
	m := testModel()
	m.thinning = true
	m.loading = true
	m.refreshing = false

	updated, cmd := m.Update(ThinResultMsg{
		Deleted:     0,
		FailedDates: []string{"2026-03-01-140100"},
		Err:         fmt.Errorf("1 snapshot deletion(s) failed: 2026-03-01-140100 (ESTALE)"),
	})
	model := updated.(Model)

	if cmd != nil {
		t.Error("expected nil command when all deletions failed (no refresh)")
	}
	if model.thinning {
		t.Error("expected thinning = false")
	}
	if _, ok := model.thinPinned["2026-03-01-140100"]; !ok {
		t.Error("expected failed date to be recorded in thinPinned")
	}
}

func TestThinResultPartialSuccessRefreshes(t *testing.T) {
	m := testModel()
	m.thinning = true
	m.loading = true
	m.refreshing = false

	updated, cmd := m.Update(ThinResultMsg{
		Deleted:     1,
		FailedDates: []string{"2026-03-01-140200"},
		Err:         fmt.Errorf("1 snapshot deletion(s) failed: 2026-03-01-140200 (ESTALE)"),
	})
	model := updated.(Model)

	if cmd == nil {
		t.Error("expected refresh command when some deletions succeeded")
	}
	if _, ok := model.thinPinned["2026-03-01-140200"]; !ok {
		t.Error("expected failed date to be recorded in thinPinned")
	}
	if model.thinning {
		t.Error("expected thinning = false after ThinResultMsg")
	}
}

func TestThinPinnedDatesFilteredFromTargets(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	// Pre-pin a date that would otherwise be a thin target.
	m.thinPinned["2026-03-01-130100"] = struct{}{}

	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-130000", Time: now.Add(-2 * time.Hour), UUID: "E3F52B7A-8C19-4D6E-A031-7F5B2E9D14C8", XID: 1547201, Purgeable: true},
		{Date: "2026-03-01-130100", Time: now.Add(-119 * time.Minute), UUID: "9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5", XID: 1547289, Purgeable: true},
	}

	updated, cmd := m.Update(RefreshResultMsg{
		Snapshots: snaps,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	// The pinned date is the only thin target, so thinning should not start.
	if model.thinning {
		t.Error("expected thinning = false when all targets are pinned")
	}
	// cmd should be nil (no thinning batch); but there might be other cmds
	// from the refresh result. Check that thinning flag is not set.
	_ = cmd
}

func TestManualRefreshClearsThinPinned(t *testing.T) {
	m := testModel()
	m.thinPinned["2026-03-01-140100"] = struct{}{}
	m.refreshing = false

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	model := updated.(Model)

	if len(model.thinPinned) != 0 {
		t.Errorf("thinPinned = %v, want empty after manual refresh", model.thinPinned)
	}
}

func TestManualRefreshClearsThinPinnedWhileRefreshing(t *testing.T) {
	m := testModel()
	m.thinPinned["2026-03-01-140100"] = struct{}{}
	m.refreshing = true

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	model := updated.(Model)

	if cmd != nil {
		t.Error("expected nil command when refresh already in flight")
	}
	if len(model.thinPinned) != 0 {
		t.Errorf("thinPinned = %v, want empty after manual refresh during in-flight refresh", model.thinPinned)
	}
	if !model.refreshPending {
		t.Error("expected refreshPending = true when manual refresh is requested in-flight")
	}
}

func TestAutoToggleOnClearsThinPinned(t *testing.T) {
	m := testModel()
	m.thinPinned["2026-03-01-140100"] = struct{}{}

	// Toggle off first (starts enabled).
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := updated.(Model)

	// Pinned should persist when toggling off.
	if len(model.thinPinned) != 1 {
		t.Errorf("thinPinned should persist when toggling auto off, got %d entries", len(model.thinPinned))
	}

	// Toggle back on: should clear pinned.
	updated, _ = model.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model = updated.(Model)

	if len(model.thinPinned) != 0 {
		t.Errorf("thinPinned = %v, want empty after toggling auto on", model.thinPinned)
	}
}

func TestUITickStopsWhenAutoDisabledAndIdle(t *testing.T) {
	m := testModel()
	// Toggle auto off (starts enabled).
	m.auto.Toggle(m.now())
	m.loading = false

	updated, cmd := m.Update(UITickMsg{})
	_ = updated.(Model)
	if cmd != nil {
		t.Error("expected nil cmd when auto is disabled and not loading")
	}
}

func TestUITickContinuesWhenAutoEnabled(t *testing.T) {
	m := testModel()
	// Auto is enabled by default in testConfig.

	updated, cmd := m.Update(UITickMsg{})
	_ = updated.(Model)
	if cmd == nil {
		t.Error("expected non-nil cmd when auto is enabled")
	}
}

func TestAutoToggleOnRestartsUITick(t *testing.T) {
	m := testModel()
	// Toggle off first.
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := updated.(Model)
	if model.auto.Enabled() {
		t.Fatal("expected auto disabled after first toggle")
	}

	// Toggle back on: should return uiTick command.
	updated, cmd := model.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model = updated.(Model)
	if !model.auto.Enabled() {
		t.Fatal("expected auto enabled after second toggle")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd (uiTick) when toggling auto on")
	}
}

func TestSuccessfulThinClearsThinPinned(t *testing.T) {
	m := testModel()
	m.thinning = true
	m.loading = true
	m.refreshing = false
	m.thinPinned["2026-03-01-140100"] = struct{}{}

	updated, _ := m.Update(ThinResultMsg{
		Deleted: 2,
		Err:     nil,
	})
	model := updated.(Model)

	if len(model.thinPinned) != 0 {
		t.Errorf("thinPinned = %v, want empty after successful thin", model.thinPinned)
	}
}

func TestFirstRefreshLogsSummaryNotIndividualAdded(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
		{Date: "2026-03-01-144000", Time: now.Add(-20 * time.Minute)},
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute)},
	}

	updated, _ := m.Update(RefreshResultMsg{
		Snapshots: snaps,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	entries := model.log.Entries()
	var foundCount, addedCount int
	for _, e := range entries {
		if e.Category == logger.CatFound {
			foundCount++
			if !strings.Contains(e.Message, "3 existing snapshots") {
				t.Errorf("FOUND message = %q, want to contain '3 existing snapshots'", e.Message)
			}
		}
		if e.Category == logger.CatAdded {
			addedCount++
		}
	}
	if foundCount != 1 {
		t.Errorf("FOUND entries = %d, want 1", foundCount)
	}
	if addedCount != 0 {
		t.Errorf("ADDED entries = %d, want 0 on first refresh", addedCount)
	}
}

func TestSecondRefreshLogsIndividualAdded(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	// First refresh: establishes baseline.
	initial := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
	}
	updated, _ := m.Update(RefreshResultMsg{
		Snapshots: initial,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	// Second refresh: new snapshot appears.
	withNew := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute)},
	}
	updated, _ = model.Update(RefreshResultMsg{
		Snapshots: withNew,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model = updated.(Model)

	entries := model.log.Entries()
	var addedCount int
	for _, e := range entries {
		if e.Category == logger.CatAdded {
			addedCount++
		}
	}
	if addedCount != 1 {
		t.Errorf("ADDED entries = %d, want 1 for new snapshot on second refresh", addedCount)
	}
}

func TestCreatedSnapshotNotDuplicatedAsAdded(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	// First refresh: establishes baseline.
	initial := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
	}
	updated, _ := m.Update(RefreshResultMsg{
		Snapshots: initial,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	// Snapshot created: records date in recentCreated.
	updated, _ = model.Update(SnapshotCreatedMsg{Date: "2026-03-01-150000"})
	model = updated.(Model)

	// Refresh includes the new snapshot.
	withNew := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
		{Date: "2026-03-01-150000", Time: now},
	}
	updated, _ = model.Update(RefreshResultMsg{
		Snapshots: withNew,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model = updated.(Model)

	entries := model.log.Entries()
	var addedCount int
	for _, e := range entries {
		if e.Category == logger.CatAdded && strings.Contains(e.Message, "2026-03-01-150000") {
			addedCount++
		}
	}
	if addedCount != 0 {
		t.Errorf("ADDED entries for created snapshot = %d, want 0 (suppressed by recentCreated)", addedCount)
	}
}

func TestThinnedSnapshotsNotDuplicatedAsRemoved(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	// First refresh: establishes baseline with two snapshots.
	initial := []snapshot.Snapshot{
		{Date: "2026-03-01-130000", Time: now.Add(-2 * time.Hour)},
		{Date: "2026-03-01-130100", Time: now.Add(-119 * time.Minute)},
	}
	updated, _ := m.Update(RefreshResultMsg{
		Snapshots: initial,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	// Thinning result: one snapshot thinned.
	updated, _ = model.Update(ThinResultMsg{
		Deleted:      1,
		ThinnedDates: []string{"2026-03-01-130100"},
	})
	model = updated.(Model)

	// Refresh: missing thinned snapshot.
	remaining := []snapshot.Snapshot{
		{Date: "2026-03-01-130000", Time: now.Add(-2 * time.Hour)},
	}
	updated, _ = model.Update(RefreshResultMsg{
		Snapshots: remaining,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model = updated.(Model)

	entries := model.log.Entries()
	var removedCount int
	for _, e := range entries {
		if e.Category == logger.CatRemoved && strings.Contains(e.Message, "2026-03-01-130100") {
			removedCount++
		}
	}
	if removedCount != 0 {
		t.Errorf("REMOVED entries for thinned snapshot = %d, want 0 (suppressed by recentThinned)", removedCount)
	}
}

func TestMaybeThinLogsThinningStart(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	// Set up snapshots old enough to trigger thinning.
	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-130000", Time: now.Add(-2 * time.Hour)},
		{Date: "2026-03-01-130100", Time: now.Add(-119 * time.Minute)},
	}
	updated, _ := m.Update(RefreshResultMsg{
		Snapshots: snaps,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	entries := model.log.Entries()
	var thinStartCount int
	for _, e := range entries {
		if e.Category == logger.CatAuto && strings.Contains(e.Message, "Thinning") {
			thinStartCount++
		}
	}
	if thinStartCount != 1 {
		t.Errorf("AUTO 'Thinning' entries = %d, want 1", thinStartCount)
	}
}

func TestDoThinSnapshotsReturnsThinnedDates(t *testing.T) {
	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil deletelocalsnapshots 2026-03-01-140000": {output: []byte("Deleted\n")},
		"tmutil deletelocalsnapshots 2026-03-01-140100": {err: fmt.Errorf("permission denied")},
	}}

	msg := doThinSnapshots(runner, []string{"2026-03-01-140000", "2026-03-01-140100"})()
	result, ok := msg.(ThinResultMsg)
	if !ok {
		t.Fatalf("msg type = %T, want ThinResultMsg", msg)
	}

	if len(result.ThinnedDates) != 1 || result.ThinnedDates[0] != "2026-03-01-140000" {
		t.Fatalf("ThinnedDates = %v, want [2026-03-01-140000]", result.ThinnedDates)
	}
}

func TestAllEstaleFailuresLogAsWarn(t *testing.T) {
	m := testModel()
	m.thinning = true
	m.loading = true
	m.refreshing = false

	updated, _ := m.Update(ThinResultMsg{
		Deleted:     0,
		FailedDates: []string{"2026-03-01-140100"},
		EstaleCount: 1,
		Err:         fmt.Errorf("1 snapshot deletion(s) failed: 2026-03-01-140100 (stale handle, skipped)"),
	})
	model := updated.(Model)

	entries := model.log.Entries()
	var warnCount, errorCount int
	for _, e := range entries {
		if e.Category == logger.CatThinned && e.Level == logger.LevelWarn {
			warnCount++
		}
		if e.Category == logger.CatThinned && e.Level == logger.LevelError {
			errorCount++
		}
	}
	if warnCount != 1 {
		t.Errorf("WARN THINNED entries = %d, want 1", warnCount)
	}
	if errorCount != 0 {
		t.Errorf("ERROR THINNED entries = %d, want 0 for all-ESTALE failures", errorCount)
	}
}

func TestMixedEstaleAndRealErrorLogsAsError(t *testing.T) {
	m := testModel()
	m.thinning = true
	m.loading = true
	m.refreshing = false

	updated, _ := m.Update(ThinResultMsg{
		Deleted:     0,
		FailedDates: []string{"2026-03-01-140100", "2026-03-01-140200"},
		EstaleCount: 1,
		Err:         fmt.Errorf("2 snapshot deletion(s) failed"),
	})
	model := updated.(Model)

	entries := model.log.Entries()
	var errorCount int
	for _, e := range entries {
		if e.Category == logger.CatThinned && e.Level == logger.LevelError {
			errorCount++
		}
	}
	if errorCount != 1 {
		t.Errorf("ERROR THINNED entries = %d, want 1 for mixed failures", errorCount)
	}
}

func TestOpenLogKeyBinding(t *testing.T) {
	m := testModel()
	m.cfg.LogDir = t.TempDir()

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	model := updated.(Model)

	if cmd == nil {
		t.Error("expected command from 'l' key press")
	}

	entries := model.log.Entries()
	var found bool
	for _, e := range entries {
		if e.Category == logger.CatOpen && strings.Contains(e.Message, "Opening log directory") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected OPEN log entry for 'l' key press")
	}
}

func TestOpenLogKeyBindingEmptyDir(t *testing.T) {
	m := testModel()
	// LogDir is "" by default in testConfig.

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	model := updated.(Model)

	if cmd != nil {
		t.Error("expected nil command when LogDir is empty")
	}

	entries := model.log.Entries()
	var found bool
	for _, e := range entries {
		if e.Category == logger.CatOpen && e.Level == logger.LevelWarn {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected WARN OPEN log entry when LogDir is empty")
	}
}

func TestRefreshSummaryOnlyLoggedOnChange(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	disk := platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"}
	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
	}

	// First refresh: should log the summary.
	updated, _ := m.Update(RefreshResultMsg{Snapshots: snaps, TMStatus: "Configured", DiskInfo: disk})
	model := updated.(Model)

	// Second refresh: same data, should not log again.
	updated, _ = model.Update(RefreshResultMsg{Snapshots: snaps, TMStatus: "Configured", DiskInfo: disk})
	model = updated.(Model)

	entries := model.log.Entries()
	var refreshCount int
	for _, e := range entries {
		if e.Category == logger.CatRefresh && strings.Contains(e.Message, "Refresh:") {
			refreshCount++
		}
	}
	if refreshCount != 1 {
		t.Errorf("REFRESH summary entries = %d, want 1 (second identical refresh should be suppressed)", refreshCount)
	}
}

func TestLogEntryAtVisualLine(t *testing.T) {
	tests := []struct {
		name       string
		entryY     []int
		totalLines int
		line       int
		want       int
	}{
		{
			name:       "first entry",
			entryY:     []int{0, 1, 2},
			totalLines: 3,
			line:       0,
			want:       0,
		},
		{
			name:       "last entry",
			entryY:     []int{0, 1, 2},
			totalLines: 3,
			line:       2,
			want:       2,
		},
		{
			name:       "wrapped entry second visual line",
			entryY:     []int{0, 3, 5},
			totalLines: 7,
			line:       4,
			want:       1,
		},
		{
			name:       "wrapped entry first visual line",
			entryY:     []int{0, 3, 5},
			totalLines: 7,
			line:       3,
			want:       1,
		},
		{
			name:       "out of range negative",
			entryY:     []int{0, 1, 2},
			totalLines: 3,
			line:       -1,
			want:       -1,
		},
		{
			name:       "out of range beyond total",
			entryY:     []int{0, 1, 2},
			totalLines: 3,
			line:       3,
			want:       -1,
		},
		{
			name:       "empty entries",
			entryY:     nil,
			totalLines: 0,
			line:       0,
			want:       -1,
		},
		{
			name:       "single entry multiple lines",
			entryY:     []int{0},
			totalLines: 4,
			line:       3,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logEntryAtVisualLine(tt.entryY, tt.totalLines, tt.line)
			if got != tt.want {
				t.Errorf("logEntryAtVisualLine(%v, %d, %d) = %d, want %d",
					tt.entryY, tt.totalLines, tt.line, got, tt.want)
			}
		})
	}
}

func TestViewPinnedIndicator(t *testing.T) {
	m := testModel()
	m.width = 120
	m.snapTable.SetWidth(contentWidth(120))
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.snapshots = []snapshot.Snapshot{
		{
			Date: "2026-03-01-145000",
			Time: now.Add(-10 * time.Minute),
			UUID: "B7C83E91-4A5D-4F12-9E68-1D3F7A2B8C04",
			XID:  1547356,
		},
		{
			Date: "2026-03-01-144000",
			Time: now.Add(-20 * time.Minute),
			UUID: "9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5",
			XID:  1547289,
		},
	}
	m.thinPinned["2026-03-01-144000"] = struct{}{}
	m.updateSnapViewContent()

	viewContent(m)

	rows := m.snapTable.Rows()
	pinnedChecked := false
	nonPinnedChecked := false
	for _, row := range rows {
		switch row[0] {
		case "2026-03-01 14:40:00":
			pinnedChecked = true
			if !strings.Contains(row[5], "pinned") {
				t.Errorf("pinned snapshot status = %q, want to contain 'pinned'", row[5])
			}
		case "2026-03-01 14:50:00":
			nonPinnedChecked = true
			if strings.Contains(row[5], "pinned") {
				t.Error("non-pinned snapshot should not show pinned indicator")
			}
		}
	}
	if !pinnedChecked {
		t.Fatal("failed to find pinned snapshot row")
	}
	if !nonPinnedChecked {
		t.Fatal("failed to find non-pinned snapshot row")
	}
}

func TestViewPinnedAndLimitsShrink(t *testing.T) {
	m := testModel()
	m.width = 120
	m.snapTable.SetWidth(contentWidth(120))
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.snapshots = []snapshot.Snapshot{
		{
			Date:         "2026-03-01-144000",
			Time:         now.Add(-20 * time.Minute),
			UUID:         "9A1D4F83-2E7B-4C05-B8F6-3D6A9E2C71F5",
			XID:          1547289,
			LimitsShrink: true,
		},
	}
	m.thinPinned["2026-03-01-144000"] = struct{}{}
	m.updateSnapViewContent()

	// Check row data directly; the rendered view may truncate the combined
	// status string at narrow terminal widths.
	rows := m.snapTable.Rows()
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	status := rows[0][5]
	if !strings.Contains(status, "pinned") {
		t.Errorf("status = %q, want to contain 'pinned'", status)
	}
	if !strings.Contains(status, "limits shrink") {
		t.Errorf("status = %q, want to contain 'limits shrink'", status)
	}
}

func TestRefreshSummaryLoggedOnDiskChange(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
	}

	// First refresh.
	updated, _ := m.Update(RefreshResultMsg{
		Snapshots: snaps, TMStatus: "Configured",
		DiskInfo: platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model := updated.(Model)

	// Second refresh: different disk info.
	updated, _ = model.Update(RefreshResultMsg{
		Snapshots: snaps, TMStatus: "Configured",
		DiskInfo: platform.DiskInfo{Total: "460Gi", Used: "220Gi", Available: "237Gi", Percent: "49%"},
	})
	model = updated.(Model)

	entries := model.log.Entries()
	var refreshCount int
	for _, e := range entries {
		if e.Category == logger.CatRefresh && strings.Contains(e.Message, "Refresh:") {
			refreshCount++
		}
	}
	if refreshCount != 2 {
		t.Errorf("REFRESH summary entries = %d, want 2 (disk changed between refreshes)", refreshCount)
	}
}

func TestFlashTickIgnoresStaleAnimationID(t *testing.T) {
	m := testModel()

	_ = m.setFocusPanel(panelLog)
	staleID := m.flash.id
	_ = m.setFocusPanel(panelInfo)

	updated, cmd := m.Update(FlashTickMsg{ID: staleID})
	model := updated.(Model)

	if model.flash.frame != 0 {
		t.Errorf("flash frame = %d, want 0 when tick ID is stale", model.flash.frame)
	}
	if cmd != nil {
		t.Error("expected nil command for stale flash tick")
	}
}

func TestFlashTickAdvancesMatchingAnimationID(t *testing.T) {
	m := testModel()

	_ = m.setFocusPanel(panelLog)
	currentID := m.flash.id

	updated, cmd := m.Update(FlashTickMsg{ID: currentID})
	model := updated.(Model)

	if model.flash.frame != 1 {
		t.Errorf("flash frame = %d, want 1 for matching tick ID", model.flash.frame)
	}
	if cmd == nil {
		t.Error("expected follow-up flash tick command for active animation")
	}
}

func TestFirstRefreshFoundCountExcludesRecentCreated(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	// Simulate user creating a snapshot before the first refresh returns.
	updated, _ := m.Update(SnapshotCreatedMsg{Date: "2026-03-01-150000"})
	model := updated.(Model)

	// First refresh includes both pre-existing and just-created snapshots.
	snaps := []snapshot.Snapshot{
		{Date: "2026-03-01-140000", Time: now.Add(-60 * time.Minute)},
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
		{Date: "2026-03-01-150000", Time: now},
	}
	updated, _ = model.Update(RefreshResultMsg{
		Snapshots: snaps,
		TMStatus:  "Configured",
		DiskInfo:  platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
	})
	model = updated.(Model)

	entries := model.log.Entries()
	var foundMsg string
	for _, e := range entries {
		if e.Category == logger.CatFound {
			foundMsg = e.Message
		}
	}

	want := "Found 2 existing snapshots"
	if foundMsg != want {
		t.Errorf("FOUND log = %q, want %q", foundMsg, want)
	}
}

func TestLogCursorStableWhenRingBufferAtCapacity(t *testing.T) {
	m := testModel()

	// Fill the ring buffer to capacity (50 entries).
	for i := range 50 {
		m.log.Log(logger.LevelInfo, logger.CatRefresh, fmt.Sprintf("entry-%d", i))
	}
	m.updateLogViewContent()

	// Move cursor to entry index 10 (which is entry-39 in newest-first display).
	m.logCursor = 10
	targetEntry := m.log.Entries()[49-10] // newest-first: index 10 = entry at position len-1-10

	// Add 3 more entries while buffer is at capacity.
	for i := range 3 {
		m.log.Log(logger.LevelInfo, logger.CatRefresh, fmt.Sprintf("new-entry-%d", i))
	}
	m.updateLogViewContent()

	// The cursor should have shifted by 3 to track the same logical entry.
	wantCursor := 13
	if m.logCursor != wantCursor {
		t.Errorf("logCursor = %d, want %d", m.logCursor, wantCursor)
	}

	// Verify the entry at the cursor position still has the same message.
	entries := m.log.Entries()
	// In newest-first display, cursor index maps to entries[len-1-cursor].
	gotIdx := len(entries) - 1 - m.logCursor
	if gotIdx >= 0 && gotIdx < len(entries) {
		if entries[gotIdx].Message != targetEntry.Message {
			t.Errorf("entry at cursor = %q, want %q", entries[gotIdx].Message, targetEntry.Message)
		}
	}
}

func TestLogViewportOffsetPreservedWhenNewEntriesArrive(t *testing.T) {
	m := testModel()
	m.width = 80
	m.height = 40

	// Add enough entries to fill the viewport and allow scrolling.
	for i := range 20 {
		m.log.Log(logger.LevelInfo, logger.CatRefresh, fmt.Sprintf("line-%d", i))
	}
	m.updateLogViewContent()

	// Scroll viewport down so YOffset > 0.
	m.logView.SetYOffset(5)
	initialOffset := m.logView.YOffset()
	if initialOffset != 5 {
		t.Fatalf("setup: YOffset = %d, want 5", initialOffset)
	}

	// Add 3 more entries, which prepend in newest-first display.
	for i := range 3 {
		m.log.Log(logger.LevelInfo, logger.CatRefresh, fmt.Sprintf("new-line-%d", i))
	}
	m.updateLogViewContent()

	// The offset should increase by the number of new visual lines (3 single-line entries).
	got := m.logView.YOffset()
	want := initialOffset + 3
	if got != want {
		t.Errorf("YOffset after new entries = %d, want %d", got, want)
	}
}

func TestUITickOnlyUpdatesAgeColumn(t *testing.T) {
	m := testModel()
	m.width = 120
	m.snapTable.SetWidth(contentWidth(120))
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.snapshots = []snapshot.Snapshot{
		{Date: "2026-03-01-145950", Time: now.Add(-10 * time.Second)},
		{Date: "2026-03-01-145955", Time: now.Add(-5 * time.Second)},
	}
	m.updateSnapViewContent()

	rowsBefore := m.snapTable.Rows()
	colsBefore := m.snapTable.Columns()
	datesBefore := make([]string, len(rowsBefore))
	agesBefore := make([]string, len(rowsBefore))
	for i, r := range rowsBefore {
		datesBefore[i] = r[0]
		agesBefore[i] = r[1]
	}

	// Advance time by 2 seconds so second-level ages change.
	m.now = func() time.Time { return now.Add(2 * time.Second) }
	m.updateSnapAges()

	rowsAfter := m.snapTable.Rows()
	colsAfter := m.snapTable.Columns()

	// Column widths should be unchanged.
	if len(colsBefore) != len(colsAfter) {
		t.Fatalf("column count changed: %d -> %d", len(colsBefore), len(colsAfter))
	}
	for i := range colsBefore {
		if colsBefore[i].Width != colsAfter[i].Width {
			t.Errorf("column %d width changed: %d -> %d", i, colsBefore[i].Width, colsAfter[i].Width)
		}
	}

	// DATE column (index 0) should be unchanged.
	for i, r := range rowsAfter {
		if r[0] != datesBefore[i] {
			t.Errorf("row %d DATE changed: %q -> %q", i, datesBefore[i], r[0])
		}
	}

	// AGE column (index 1) should have updated values.
	for i, r := range rowsAfter {
		if r[1] == agesBefore[i] {
			t.Errorf("row %d AGE did not update after tick: still %q", i, r[1])
		}
	}
}

func TestTidemarkFetchFailureLogged(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	updated, _ := m.Update(RefreshResultMsg{
		Snapshots:   []snapshot.Snapshot{{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)}},
		TMStatus:    "Configured",
		DiskInfo:    platform.DiskInfo{Total: "460Gi", Used: "215Gi", Available: "242Gi", Percent: "48%"},
		TidemarkErr: fmt.Errorf("container not found"),
	})
	model := updated.(Model)

	entries := model.log.Entries()
	var found bool
	for _, e := range entries {
		if e.Level == logger.LevelWarn && strings.Contains(e.Message, "Tidemark fetch failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected WARN log for tidemark fetch failure")
	}
}
