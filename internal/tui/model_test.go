package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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
	log := logger.New("", 50)
	runner := &mockRunner{responses: map[string]mockResponse{}}
	m := NewModel(cfg, runner, log, "", "Configured", "dev")
	m.width = 80
	m.height = 40
	return m
}

func TestViewEmpty(t *testing.T) {
	m := testModel()
	v := m.View()

	if !strings.Contains(v, "SNAPPY") {
		t.Error("view missing SNAPPY title")
	}
	if !strings.Contains(v, "LOCAL SNAPSHOTS (0)") {
		t.Error("view missing snapshot count")
	}
	if !strings.Contains(v, "press 's'") {
		t.Error("view missing empty state hint")
	}
	if !strings.Contains(v, "[s]") {
		t.Error("view missing controls")
	}
}

func TestViewWithSnapshots(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.snapshots = []snapshot.Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
		{Date: "2026-03-01-144000", Time: now.Add(-20 * time.Minute)},
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute)},
	}

	v := m.View()

	if !strings.Contains(v, "LOCAL SNAPSHOTS (3)") {
		t.Error("view missing correct snapshot count")
	}
	if !strings.Contains(v, "2026-03-01-145000") {
		t.Error("view missing newest snapshot")
	}
	if !strings.Contains(v, "2026-03-01-143000") {
		t.Error("view missing oldest snapshot")
	}
}

func TestViewBookend(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	var snaps []snapshot.Snapshot
	for i := range 6 {
		d := now.Add(-time.Duration(60-i*10) * time.Minute)
		snaps = append(snaps, snapshot.Snapshot{
			Date: d.Format("2006-01-02-150405"),
			Time: d,
		})
	}
	m.snapshots = snaps

	v := m.View()

	if !strings.Contains(v, "LOCAL SNAPSHOTS (6)") {
		t.Error("view missing correct snapshot count")
	}
	if !strings.Contains(v, "... and 2 more ...") {
		t.Error("view missing bookend ellipsis")
	}
}

func TestViewQuitting(t *testing.T) {
	m := testModel()
	m.quitting = true
	v := m.View()
	if v != "" {
		t.Errorf("View() when quitting = %q, want empty", v)
	}
}

func TestKeyQuit(t *testing.T) {
	m := testModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
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
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model := updated.(Model)
	if model.auto.Enabled() {
		t.Error("expected auto disabled after toggle")
	}

	// Toggle back on
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
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
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute)},
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
		{Date: "2026-03-01-145000", Time: now.Add(-10 * time.Minute)},
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
		if e.Type == logger.Removed {
			removedCount++
		}
		if e.Type == logger.Error && strings.Contains(e.Message, "Failed to list snapshots") {
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
		if e.Type == logger.Error {
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
	v := m.View()
	if !strings.Contains(v, "on") {
		t.Error("view should show auto-snapshot 'on'")
	}
}

func TestViewAutoStatusOff(t *testing.T) {
	m := testModel()
	m.auto.Toggle(m.now())
	v := m.View()
	if !strings.Contains(v, "off") {
		t.Error("view should show auto-snapshot 'off'")
	}
}

func TestViewAPFSDetails(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }
	m.apfsVolume = "disk3s5"
	m.snapshots = []snapshot.Snapshot{
		{
			Date:         "2026-03-01-145000",
			Time:         now.Add(-10 * time.Minute),
			UUID:         "ABC-123",
			Purgeable:    true,
			LimitsShrink: false,
		},
		{
			Date:         "2026-03-01-144000",
			Time:         now.Add(-20 * time.Minute),
			UUID:         "DEF-456",
			Purgeable:    false,
			LimitsShrink: true,
		},
	}

	v := m.View()

	if !strings.Contains(v, "APFS Volume: disk3s5") {
		t.Error("view missing APFS volume")
	}
	if !strings.Contains(v, "ABC-123") {
		t.Error("view missing UUID for first snapshot")
	}
	if !strings.Contains(v, "purgeable") {
		t.Error("view missing purgeable flag")
	}
}

func TestExactlyFourSnapshots(t *testing.T) {
	m := testModel()
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	m.now = func() time.Time { return now }

	for i := range 4 {
		d := now.Add(-time.Duration(40-i*10) * time.Minute)
		m.snapshots = append(m.snapshots, snapshot.Snapshot{
			Date: d.Format("2006-01-02-150405"),
			Time: d,
		})
	}

	v := m.View()
	if strings.Contains(v, "... and") {
		t.Error("exactly 4 snapshots should not show ellipsis")
	}
}

func TestDiffDisplay(t *testing.T) {
	m := testModel()
	m.diffAdded = 2
	m.diffRemoved = 1
	v := m.View()
	if !strings.Contains(v, "+2 added") {
		t.Error("view missing diff added count")
	}
	if !strings.Contains(v, "1 removed") {
		t.Error("view missing diff removed count")
	}
}

func TestSnapshotKeyIgnoredWhileSnapshotting(t *testing.T) {
	m := testModel()
	m.snapshotting = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
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
	m.refreshing = false

	updated, _ := m.Update(ThinResultMsg{Deleted: 1})
	model := updated.(Model)

	if model.thinning {
		t.Error("expected thinning = false after ThinResultMsg")
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
}
