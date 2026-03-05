package snapshot

import (
	"time"
)

// AutoManager controls automatic snapshot creation and thinning.
type AutoManager struct {
	enabled      bool
	interval     time.Duration
	thinAge      time.Duration
	thinCadence  time.Duration
	lastAutoTime time.Time
}

// NewAutoManager creates an AutoManager with the given configuration.
// The timer starts from now, so the first auto-snapshot fires after one
// full interval.
func NewAutoManager(enabled bool, interval, thinAge, thinCadence time.Duration, now time.Time) *AutoManager {
	return &AutoManager{
		enabled:      enabled,
		interval:     interval,
		thinAge:      thinAge,
		thinCadence:  thinCadence,
		lastAutoTime: now,
	}
}

// Enabled returns whether auto-snapshots are currently active.
func (a *AutoManager) Enabled() bool {
	return a.enabled
}

// ShouldSnapshot returns true if the auto-snapshot interval has elapsed.
func (a *AutoManager) ShouldSnapshot(now time.Time) bool {
	if !a.enabled {
		return false
	}
	return now.Sub(a.lastAutoTime) >= a.interval
}

// RecordSnapshot advances the timer. Call this after attempting a snapshot,
// regardless of whether it succeeded (matching bash behavior).
func (a *AutoManager) RecordSnapshot(now time.Time) {
	a.lastAutoTime = now
}

// Toggle flips the enabled state. When enabling, the timer resets to now.
// Returns the new enabled state.
func (a *AutoManager) Toggle(now time.Time) bool {
	a.enabled = !a.enabled
	if a.enabled {
		a.lastAutoTime = now
	}
	return a.enabled
}

// NextIn returns the duration until the next auto-snapshot, or zero
// if disabled or overdue.
func (a *AutoManager) NextIn(now time.Time) time.Duration {
	if !a.enabled {
		return 0
	}
	remaining := a.interval - now.Sub(a.lastAutoTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Interval returns the configured auto-snapshot interval.
func (a *AutoManager) Interval() time.Duration {
	return a.interval
}

// ThinAge returns the age threshold for thinning.
func (a *AutoManager) ThinAge() time.Duration {
	return a.thinAge
}

// ThinCadence returns the thinning cadence.
func (a *AutoManager) ThinCadence() time.Duration {
	return a.thinCadence
}

// ComputeThinTargets walks snapshots oldest-first, applying the age
// threshold and cadence to determine which dates should be deleted.
// The input snapshots must be sorted ascending by date.
//
// Snapshots whose dates appear in pinned are treated as kept (their time
// updates lastKeptTime) but are never added to the deletion targets. This
// preserves even cadence spacing around snapshots the system refuses to
// delete (e.g., ESTALE from a stale kernel handle).
func (a *AutoManager) ComputeThinTargets(snapshots []Snapshot, now time.Time, pinned map[string]struct{}) []string {
	if !a.enabled || len(snapshots) == 0 {
		return nil
	}

	var lastKeptTime time.Time
	var targets []string

	for _, snap := range snapshots {
		age := now.Sub(snap.Time)
		if age < a.thinAge {
			continue
		}

		if lastKeptTime.IsZero() {
			lastKeptTime = snap.Time
			continue
		}

		// Pinned snapshots act as cadence anchors but are never deleted.
		if _, ok := pinned[snap.Date]; ok {
			lastKeptTime = snap.Time
			continue
		}

		gap := snap.Time.Sub(lastKeptTime)
		if gap >= a.thinCadence {
			lastKeptTime = snap.Time
			continue
		}

		targets = append(targets, snap.Date)
	}

	return targets
}
