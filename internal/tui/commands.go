package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

func doRefresh(runner platform.CommandRunner, cfg *config.Config, apfsVolume string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tmStatus := platform.CheckStatus(ctx, runner)

		dates, err := platform.ListSnapshots(ctx, runner, cfg.MountPoint)
		if err != nil {
			diskInfo, diskErr := platform.GetDiskInfo(ctx, runner, cfg.MountPoint)
			return RefreshResultMsg{
				TMStatus:    tmStatus,
				DiskInfo:    diskInfo,
				DiskErr:     diskErr != nil,
				SnapshotErr: err,
			}
		}

		var snapshots []snapshot.Snapshot
		for _, d := range dates {
			t, parseErr := snapshot.ParseDate(d)
			if parseErr != nil {
				continue
			}
			snapshots = append(snapshots, snapshot.Snapshot{Date: d, Time: t})
		}

		var apfsInfo platform.APFSInfo
		var apfsErr error
		if apfsVolume != "" {
			details, otherCount, detailErr := platform.GetSnapshotDetails(ctx, runner, apfsVolume)
			if detailErr != nil {
				apfsErr = detailErr
			} else {
				apfsInfo = platform.APFSInfo{
					Volume:         apfsVolume,
					Details:        details,
					OtherSnapCount: otherCount,
				}
				// Merge APFS details into snapshots
				for i, s := range snapshots {
					if d, ok := details[s.Date]; ok {
						snapshots[i].UUID = d.UUID
						snapshots[i].Purgeable = d.Purgeable
						snapshots[i].LimitsShrink = d.LimitsShrink
					}
				}
			}
		}

		diskInfo, diskErr := platform.GetDiskInfo(ctx, runner, cfg.MountPoint)

		return RefreshResultMsg{
			Snapshots:   snapshots,
			TMStatus:    tmStatus,
			APFSInfo:    apfsInfo,
			DiskInfo:    diskInfo,
			DiskErr:     diskErr != nil,
			SnapshotErr: nil,
			APFSErr:     apfsErr,
		}
	}
}

func doCreateSnapshot(runner platform.CommandRunner) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		date, err := platform.CreateSnapshot(ctx, runner)
		return SnapshotCreatedMsg{Date: date, Err: err}
	}
}

func doThinSnapshots(runner platform.CommandRunner, targets []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		deleted := 0
		var failed []string
		for _, date := range targets {
			if err := platform.DeleteSnapshot(ctx, runner, date); err != nil {
				failed = append(failed, fmt.Sprintf("%s (%v)", date, err))
				continue
			}
			deleted++
		}

		var err error
		if len(failed) > 0 {
			err = fmt.Errorf("%d snapshot deletion(s) failed: %s", len(failed), strings.Join(failed, "; "))
		}

		return ThinResultMsg{Deleted: deleted, Err: err}
	}
}

func refreshTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}
