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
		MountPoint:           "/",
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
	m := NewModel(cfg, runner, log, "disk3s5", "Configured", "/", "dev")
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

func TestViewAPFSDetails(t *testing.T) {
	m := testModel()
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

func TestMouseClickSnapshotSelectsTopVisibleRowWhenTableIsOffset(t *testing.T) {
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

	for range 2 {
		updated, _ = model.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
		model = updated.(Model)
	}

	if got := model.snapTable.Cursor(); got != 2 {
		t.Fatalf("cursor before click = %d, want 2", got)
	}

	updated, _ = model.Update(tea.MouseClickMsg{
		Button: tea.MouseLeft,
		Y:      model.snapPanelY + 2,
	})
	model = updated.(Model)

	if got := model.snapTable.Cursor(); got != 1 {
		t.Fatalf("cursor after click = %d, want 1 (top visible row)", got)
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
