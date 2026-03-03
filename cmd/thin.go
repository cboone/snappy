package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

var thinCmd = &cobra.Command{
	Use:   "thin",
	Short: "Thin old snapshots based on configured cadence",
	Args:  cobra.NoArgs,
	RunE:  runThin,
}

func init() {
	thinCmd.Flags().Bool("json", false, "output in JSON format")
	rootCmd.AddCommand(thinCmd)
}

func runThin(cmd *cobra.Command, _ []string) error {
	if err := requireTmutil(); err != nil {
		return err
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	cfg := config.Load()
	runner := newRunner()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	snapshots, _, _, err := loadSnapshots(ctx, runner, cfg)
	if err != nil {
		return err
	}

	now := time.Now()
	mgr := snapshot.NewAutoManager(true, cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence, now)
	targets := mgr.ComputeThinTargets(snapshots, now)

	if len(targets) == 0 {
		if jsonOut {
			return writeJSON(cmd.OutOrStdout(), struct {
				Thinned int `json:"thinned"`
			}{Thinned: 0})
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "No snapshots to thin")
		return err
	}

	deleted, deleteErr := deleteThinTargets(runner, targets)

	w := cmd.OutOrStdout()
	if jsonOut {
		if err := writeJSON(w, struct {
			Thinned int `json:"thinned"`
		}{Thinned: deleted}); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "Thinned %d snapshot(s)\n", deleted); err != nil {
			return err
		}
	}

	return deleteErr
}

func deleteThinTargets(runner platform.CommandRunner, targets []string) (int, error) {
	deleted := 0
	var failed []string

	for _, date := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := platform.DeleteSnapshot(ctx, runner, date)
		cancel()

		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", date, err))
			continue
		}
		deleted++
	}

	if len(failed) > 0 {
		return deleted, fmt.Errorf("%d snapshot deletion(s) failed: %s", len(failed), strings.Join(failed, "; "))
	}

	return deleted, nil
}
