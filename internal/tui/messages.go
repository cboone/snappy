package tui

import (
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

// RefreshTickMsg signals that the periodic refresh timer has fired.
type RefreshTickMsg struct{}

// RefreshResultMsg carries the result of a full data refresh.
type RefreshResultMsg struct {
	Snapshots []snapshot.Snapshot
	TMStatus  string
	APFSInfo  platform.APFSInfo
	DiskInfo  platform.DiskInfo
	DiskErr   bool
}

// SnapshotCreatedMsg signals that a snapshot creation attempt completed.
type SnapshotCreatedMsg struct {
	Date string
	Err  error
}

// ThinResultMsg signals that a thinning operation completed.
type ThinResultMsg struct {
	Deleted int
	Err     error
}
