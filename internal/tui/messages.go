package tui

import (
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

// UITickMsg drives a 1-second UI refresh so the countdown timer stays current.
type UITickMsg struct{}

// RefreshTickMsg signals that the periodic refresh timer has fired.
type RefreshTickMsg struct{}

// RefreshResultMsg carries the result of a full data refresh.
type RefreshResultMsg struct {
	Snapshots   []snapshot.Snapshot
	TMStatus    string
	APFSInfo    platform.APFSInfo
	DiskInfo    platform.DiskInfo
	DiskErr     bool
	SnapshotErr error
	APFSErr     error
	Tidemark    int64
}

// SnapshotCreatedMsg signals that a snapshot creation attempt completed.
type SnapshotCreatedMsg struct {
	Date string
	Err  error
}

// ThinResultMsg signals that a thinning operation completed.
type ThinResultMsg struct {
	Deleted     int
	FailedDates []string
	Err         error
}
