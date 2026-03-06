package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Time Machine and disk status",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().Bool("json", false, "output in JSON format")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, _ []string) error {
	if err := requireTmutil(); err != nil {
		return err
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	cfg := config.Load()
	runner := newRunner()

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	tmStatus := platform.CheckStatus(ctx, runner)
	apfsVolume, _ := platform.FindAPFSVolume(ctx, runner, config.DefaultMount)
	diskInfo, diskErr := platform.GetDiskInfo(ctx, runner, config.DefaultMount)

	dates, snapErr := platform.ListSnapshots(ctx, runner, config.DefaultMount)
	localCount := len(dates)

	var otherCount int
	if apfsVolume != "" {
		_, otherCount, _ = platform.GetSnapshotDetails(ctx, runner, apfsVolume)
	}

	if jsonOut {
		return writeStatusJSON(cmd, cfg, tmStatus, apfsVolume, diskInfo, diskErr, localCount, otherCount, snapErr)
	}

	return writeStatusHuman(cmd, cfg, tmStatus, apfsVolume, diskInfo, diskErr, localCount, otherCount, snapErr)
}

func writeStatusJSON(cmd *cobra.Command, cfg *config.Config, tmStatus, apfsVolume string, diskInfo platform.DiskInfo, diskErr error, localCount, otherCount int, snapErr error) error {
	type diskJSON struct {
		Total     string `json:"total"`
		Used      string `json:"used"`
		Available string `json:"available"`
		Percent   string `json:"percent"`
	}

	type autoJSON struct {
		Enabled  bool   `json:"enabled"`
		Interval string `json:"interval"`
		ThinAge  string `json:"thin_age"`
		ThinGap  string `json:"thin_gap"`
	}

	type snapshotsJSON struct {
		Local int    `json:"local"`
		Other int    `json:"other"`
		Error string `json:"error,omitempty"`
	}

	var disk *diskJSON
	if diskErr == nil {
		disk = &diskJSON{
			Total:     diskInfo.Total,
			Used:      diskInfo.Used,
			Available: diskInfo.Available,
			Percent:   diskInfo.Percent,
		}
	}

	snaps := snapshotsJSON{Local: localCount, Other: otherCount}
	if snapErr != nil {
		snaps.Error = snapErr.Error()
	}

	return writeJSON(cmd.OutOrStdout(), struct {
		TimeMachine string        `json:"time_machine"`
		Mount       string        `json:"mount"`
		APFSVolume  string        `json:"apfs_volume,omitempty"`
		Disk        *diskJSON     `json:"disk,omitempty"`
		Snapshots   snapshotsJSON `json:"snapshots"`
		Auto        autoJSON      `json:"auto"`
	}{
		TimeMachine: tmStatus,
		Mount:       config.DefaultMount,
		APFSVolume:  apfsVolume,
		Disk:        disk,
		Snapshots:   snaps,
		Auto: autoJSON{
			Enabled:  cfg.AutoEnabled,
			Interval: cfg.AutoSnapshotInterval.String(),
			ThinAge:  cfg.ThinAgeThreshold.String(),
			ThinGap:  cfg.ThinCadence.String(),
		},
	})
}

func writeStatusHuman(cmd *cobra.Command, cfg *config.Config, tmStatus, apfsVolume string, diskInfo platform.DiskInfo, diskErr error, localCount, otherCount int, snapErr error) error {
	w := cmd.OutOrStdout()

	if _, err := fmt.Fprintf(w, "Time Machine: %s\n", tmStatus); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Mount: %s\n", config.DefaultMount); err != nil {
		return err
	}
	if apfsVolume != "" {
		if _, err := fmt.Fprintf(w, "APFS volume: %s\n", apfsVolume); err != nil {
			return err
		}
	}
	if diskErr == nil {
		if _, err := fmt.Fprintf(w, "Disk: %s\n", diskInfo.String()); err != nil {
			return err
		}
	}
	if snapErr != nil {
		if _, err := fmt.Fprintf(w, "Snapshots: unknown (%v)\n", snapErr); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "Snapshots: %d local, %d other\n", localCount, otherCount); err != nil {
			return err
		}
	}

	autoStatus := "disabled"
	if cfg.AutoEnabled {
		autoStatus = fmt.Sprintf("enabled (every %s, thin >%s to %s)",
			cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence)
	}
	_, err := fmt.Fprintf(w, "Auto-snapshot: %s\n", autoStatus)
	return err
}
