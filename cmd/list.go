package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/snapshot"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local snapshots with details",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	listCmd.Flags().Bool("json", false, "output in JSON format")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	if err := requireTmutil(); err != nil {
		return err
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	runner := newRunner()

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	snapshots, _, _, err := loadSnapshots(ctx, runner)
	if err != nil {
		return err
	}

	if jsonOut {
		return writeListJSON(cmd, snapshots)
	}

	return writeListHuman(cmd, snapshots)
}

func writeListJSON(cmd *cobra.Command, snapshots []snapshot.Snapshot) error {
	now := time.Now()

	type jsonSnapshot struct {
		Date         string `json:"date"`
		Relative     string `json:"relative"`
		UUID         string `json:"uuid,omitempty"`
		Purgeable    *bool  `json:"purgeable,omitempty"`
		LimitsShrink *bool  `json:"limits_shrink,omitempty"`
	}

	items := make([]jsonSnapshot, len(snapshots))
	for i, s := range snapshots {
		item := jsonSnapshot{
			Date:     s.Date,
			Relative: snapshot.FormatRelativeTime(s.Time, now),
			UUID:     s.UUID,
		}
		if s.UUID != "" {
			item.Purgeable = &s.Purgeable
			item.LimitsShrink = &s.LimitsShrink
		}
		items[i] = item
	}

	return writeJSON(cmd.OutOrStdout(), struct {
		Mount     string         `json:"mount"`
		Count     int            `json:"count"`
		Snapshots []jsonSnapshot `json:"snapshots"`
	}{
		Mount:     config.DefaultMount,
		Count:     len(snapshots),
		Snapshots: items,
	})
}

func writeListHuman(cmd *cobra.Command, snapshots []snapshot.Snapshot) error {
	w := cmd.OutOrStdout()
	count := len(snapshots)
	now := time.Now()

	if _, err := fmt.Fprintf(w, "%d snapshot(s) on %s\n", count, config.DefaultMount); err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	// Display newest first (reverse of ascending loadSnapshots order).
	for i := count - 1; i >= 0; i-- {
		s := snapshots[i]
		relative := snapshot.FormatRelativeTime(s.Time, now)
		num := count - i

		line := fmt.Sprintf("  %2d. %s   (%s)", num, s.Date, relative)

		if s.UUID != "" {
			flags := "purgeable"
			if !s.Purgeable {
				flags = "pinned"
			}
			if s.LimitsShrink {
				flags += "   limits shrink"
			}
			line += fmt.Sprintf("   %s   %s", s.UUID, flags)
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}
