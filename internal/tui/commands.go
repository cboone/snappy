package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

func doRefresh(runner platform.CommandRunner, apfsVolume, apfsContainer string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tmStatus := platform.CheckStatus(ctx, runner)

		dates, err := platform.ListSnapshots(ctx, runner, config.DefaultMount)
		if err != nil {
			diskInfo, diskErr := platform.GetDiskInfo(ctx, runner, config.DefaultMount)
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
			apfsInfo.Volume = apfsVolume

			details, otherCount, detailErr := platform.GetSnapshotDetails(ctx, runner, apfsVolume)
			if detailErr != nil {
				apfsErr = detailErr
			} else {
				apfsInfo.Details = details
				apfsInfo.OtherSnapCount = otherCount
				// Merge APFS details into snapshots
				for i, s := range snapshots {
					if d, ok := details[s.Date]; ok {
						snapshots[i].UUID = d.UUID
						snapshots[i].XID = d.XID
						snapshots[i].Purgeable = d.Purgeable
						snapshots[i].LimitsShrink = d.LimitsShrink
					}
				}
			}
		}

		diskInfo, diskErr := platform.GetDiskInfo(ctx, runner, config.DefaultMount)

		var tidemark int64
		if apfsContainer != "" {
			tm, tmErr := platform.GetContainerTidemark(ctx, runner, apfsContainer)
			if tmErr == nil {
				tidemark = tm
			}
		}

		return RefreshResultMsg{
			Snapshots:   snapshots,
			TMStatus:    tmStatus,
			APFSInfo:    apfsInfo,
			DiskInfo:    diskInfo,
			DiskErr:     diskErr != nil,
			SnapshotErr: nil,
			APFSErr:     apfsErr,
			Tidemark:    tidemark,
		}
	}
}

func doCreateSnapshot(runner platform.CommandRunner) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		date, err := platform.CreateSnapshot(ctx, runner)
		return SnapshotCreatedMsg{Date: date, Err: err}
	}
}

func doThinSnapshots(runner platform.CommandRunner, targets []string) tea.Cmd {
	return func() tea.Msg {
		deleted := 0
		estaleCount := 0
		var thinnedDates []string
		var failedDates []string
		var failedDetails []string
		for _, date := range targets {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := platform.DeleteSnapshot(ctx, runner, date)
			cancel()
			if err != nil {
				failedDates = append(failedDates, date)
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) && exitErr.ExitCode() == 70 {
					estaleCount++
					failedDetails = append(failedDetails, fmt.Sprintf("%s (stale handle, skipped)", date))
				} else {
					failedDetails = append(failedDetails, fmt.Sprintf("%s (%v)", date, err))
				}
				continue
			}
			deleted++
			thinnedDates = append(thinnedDates, date)
		}

		var err error
		if len(failedDetails) > 0 {
			err = fmt.Errorf("%d snapshot deletion(s) failed: %s", len(failedDetails), strings.Join(failedDetails, "; "))
		}

		return ThinResultMsg{
			Deleted:      deleted,
			ThinnedDates: thinnedDates,
			FailedDates:  failedDates,
			EstaleCount:  estaleCount,
			Err:          err,
		}
	}
}

func refreshTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return RefreshTickMsg{}
	})
}

func uiTick() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return UITickMsg{}
	})
}

func doOpenLogDir(dir string) tea.Cmd {
	return func() tea.Msg {
		if dir == "" {
			return OpenLogDirResultMsg{Err: fmt.Errorf("log directory path is empty")}
		}

		if err := exec.Command("open", dir).Start(); err != nil {
			return OpenLogDirResultMsg{Err: err}
		}

		return OpenLogDirResultMsg{}
	}
}
