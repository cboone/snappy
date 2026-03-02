package snapshot

import (
	"testing"
	"time"
)

func TestShouldSnapshot(t *testing.T) {
	start := time.Date(2026, 3, 1, 14, 0, 0, 0, time.Local)
	am := NewAutoManager(true, 60*time.Second, 600*time.Second, 300*time.Second, start)

	if am.ShouldSnapshot(start.Add(30 * time.Second)) {
		t.Error("ShouldSnapshot() = true after 30s, want false")
	}
	if !am.ShouldSnapshot(start.Add(60 * time.Second)) {
		t.Error("ShouldSnapshot() = false after 60s, want true")
	}
	if !am.ShouldSnapshot(start.Add(120 * time.Second)) {
		t.Error("ShouldSnapshot() = false after 120s, want true")
	}
}

func TestShouldSnapshotDisabled(t *testing.T) {
	start := time.Date(2026, 3, 1, 14, 0, 0, 0, time.Local)
	am := NewAutoManager(false, 60*time.Second, 600*time.Second, 300*time.Second, start)

	if am.ShouldSnapshot(start.Add(120 * time.Second)) {
		t.Error("ShouldSnapshot() = true when disabled")
	}
}

func TestRecordSnapshot(t *testing.T) {
	start := time.Date(2026, 3, 1, 14, 0, 0, 0, time.Local)
	am := NewAutoManager(true, 60*time.Second, 600*time.Second, 300*time.Second, start)

	now := start.Add(65 * time.Second)
	am.RecordSnapshot(now)

	// Timer resets: should not fire for another 60s
	if am.ShouldSnapshot(now.Add(30 * time.Second)) {
		t.Error("ShouldSnapshot() = true 30s after record, want false")
	}
	if !am.ShouldSnapshot(now.Add(60 * time.Second)) {
		t.Error("ShouldSnapshot() = false 60s after record, want true")
	}
}

func TestToggle(t *testing.T) {
	start := time.Date(2026, 3, 1, 14, 0, 0, 0, time.Local)
	am := NewAutoManager(true, 60*time.Second, 600*time.Second, 300*time.Second, start)

	// Toggle off
	now := start.Add(30 * time.Second)
	got := am.Toggle(now)
	if got {
		t.Error("Toggle() = true, want false (toggling off)")
	}
	if am.Enabled() {
		t.Error("Enabled() = true after Toggle to off")
	}

	// Toggle back on: timer should reset
	now = now.Add(30 * time.Second)
	got = am.Toggle(now)
	if !got {
		t.Error("Toggle() = false, want true (toggling on)")
	}
	// Should not fire immediately
	if am.ShouldSnapshot(now.Add(30 * time.Second)) {
		t.Error("ShouldSnapshot() = true 30s after re-enable")
	}
	if !am.ShouldSnapshot(now.Add(60 * time.Second)) {
		t.Error("ShouldSnapshot() = false 60s after re-enable")
	}
}

func TestNextIn(t *testing.T) {
	start := time.Date(2026, 3, 1, 14, 0, 0, 0, time.Local)
	am := NewAutoManager(true, 60*time.Second, 600*time.Second, 300*time.Second, start)

	got := am.NextIn(start.Add(20 * time.Second))
	if got != 40*time.Second {
		t.Errorf("NextIn() = %v, want 40s", got)
	}

	// Overdue returns 0
	got = am.NextIn(start.Add(90 * time.Second))
	if got != 0 {
		t.Errorf("NextIn() = %v, want 0 when overdue", got)
	}

	// Disabled returns 0
	am.Toggle(start)
	got = am.NextIn(start.Add(20 * time.Second))
	if got != 0 {
		t.Errorf("NextIn() = %v, want 0 when disabled", got)
	}
}

func TestComputeThinTargetsDisabled(t *testing.T) {
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	am := NewAutoManager(false, 60*time.Second, 600*time.Second, 300*time.Second, now)

	snapshots := []Snapshot{
		{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
		{Date: "2026-03-01-143100", Time: now.Add(-29 * time.Minute)},
	}

	targets := am.ComputeThinTargets(snapshots, now)
	if targets != nil {
		t.Errorf("ComputeThinTargets() = %v, want nil when disabled", targets)
	}
}

func TestComputeThinTargets(t *testing.T) {
	now := time.Date(2026, 3, 1, 15, 0, 0, 0, time.Local)
	am := NewAutoManager(true, 60*time.Second, 600*time.Second, 300*time.Second, now)

	tests := []struct {
		name      string
		snapshots []Snapshot
		wantCount int
		wantDates []string
	}{
		{
			name:      "empty",
			snapshots: nil,
			wantCount: 0,
		},
		{
			name: "all young, none thinned",
			snapshots: []Snapshot{
				{Date: "2026-03-01-145500", Time: now.Add(-5 * time.Minute)},
				{Date: "2026-03-01-145600", Time: now.Add(-4 * time.Minute)},
				{Date: "2026-03-01-145700", Time: now.Add(-3 * time.Minute)},
			},
			wantCount: 0,
		},
		{
			name: "old snapshots spaced apart, none thinned",
			snapshots: []Snapshot{
				{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
				{Date: "2026-03-01-143500", Time: now.Add(-25 * time.Minute)},
				{Date: "2026-03-01-144000", Time: now.Add(-20 * time.Minute)},
			},
			wantCount: 0,
		},
		{
			name: "old snapshots too close together",
			snapshots: []Snapshot{
				{Date: "2026-03-01-143000", Time: now.Add(-30 * time.Minute)},
				{Date: "2026-03-01-143100", Time: now.Add(-29 * time.Minute)},
				{Date: "2026-03-01-143200", Time: now.Add(-28 * time.Minute)},
				{Date: "2026-03-01-143500", Time: now.Add(-25 * time.Minute)},
			},
			wantCount: 2,
			wantDates: []string{"2026-03-01-143100", "2026-03-01-143200"},
		},
		{
			name: "mix of old and young",
			snapshots: []Snapshot{
				{Date: "2026-03-01-142000", Time: now.Add(-40 * time.Minute)},
				{Date: "2026-03-01-142100", Time: now.Add(-39 * time.Minute)},
				{Date: "2026-03-01-145500", Time: now.Add(-5 * time.Minute)},
				{Date: "2026-03-01-145600", Time: now.Add(-4 * time.Minute)},
			},
			wantCount: 1,
			wantDates: []string{"2026-03-01-142100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := am.ComputeThinTargets(tt.snapshots, now)
			if len(targets) != tt.wantCount {
				t.Errorf("ComputeThinTargets() = %d targets, want %d: %v", len(targets), tt.wantCount, targets)
			}
			for i, d := range tt.wantDates {
				if i < len(targets) && targets[i] != d {
					t.Errorf("targets[%d] = %q, want %q", i, targets[i], d)
				}
			}
		})
	}
}
