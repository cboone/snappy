package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

func doRefresh(runner platform.CommandRunner, cfg *config.Config, apfsVolume string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		dates, err := platform.ListSnapshots(ctx, runner, cfg.MountPoint)
		if err != nil {
			return RefreshResultMsg{DiskErr: true}
		}

		var snapshots []snapshot.Snapshot
		for _, d := range dates {
			t, parseErr := snapshot.ParseDate(d)
			if parseErr != nil {
				continue
			}
			snapshots = append(snapshots, snapshot.Snapshot{Date: d, Time: t})
		}

		tmStatus := platform.CheckStatus(ctx, runner)

		var apfsInfo platform.APFSInfo
		if apfsVolume != "" {
			details, otherCount, detailErr := platform.GetSnapshotDetails(ctx, runner, apfsVolume)
			if detailErr == nil {
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
			Snapshots: snapshots,
			TMStatus:  tmStatus,
			APFSInfo:  apfsInfo,
			DiskInfo:  diskInfo,
			DiskErr:   diskErr != nil,
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
		for _, date := range targets {
			if err := platform.DeleteSnapshot(ctx, runner, date); err == nil {
				deleted++
			}
		}
		return ThinResultMsg{Deleted: deleted}
	}
}

func refreshTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}
