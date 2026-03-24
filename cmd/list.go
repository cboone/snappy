package cmd

import (
	"context"
	"fmt"
	"strings"
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
	listCmd.Flags().Bool("no-header", false, "suppress column header row")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	if err := requireTmutil(); err != nil {
		return err
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	noHeader, _ := cmd.Flags().GetBool("no-header")
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

	return writeListHuman(cmd, snapshots, noHeader)
}

func writeListJSON(cmd *cobra.Command, snapshots []snapshot.Snapshot) error {
	now := time.Now()

	type jsonSnapshot struct {
		Date         string `json:"date"`
		Relative     string `json:"relative"`
		UUID         string `json:"uuid,omitempty"`
		XIDDelta     *int   `json:"xid_delta,omitempty"`
		Purgeable    *bool  `json:"purgeable,omitempty"`
		LimitsShrink *bool  `json:"limits_shrink,omitempty"`
	}

	items := make([]jsonSnapshot, len(snapshots))
	for i, s := range snapshots {
		relative := formatRelativeAgo(s.Time, now)
		item := jsonSnapshot{
			Date:     s.Date,
			Relative: relative,
			UUID:     s.UUID,
		}
		if s.UUID != "" {
			item.Purgeable = &s.Purgeable
			item.LimitsShrink = &s.LimitsShrink
			if i > 0 && snapshots[i-1].UUID != "" {
				delta := s.XID - snapshots[i-1].XID
				item.XIDDelta = &delta
			}
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

func writeListHuman(cmd *cobra.Command, snapshots []snapshot.Snapshot, noHeader bool) error {
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

	// Check whether any snapshot has APFS details.
	hasAPFS := false
	for _, s := range snapshots {
		if s.UUID != "" {
			hasAPFS = true
			break
		}
	}

	ct := listTable(hasAPFS)

	// Display newest first (reverse of ascending loadSnapshots order).
	for i := count - 1; i >= 0; i-- {
		s := snapshots[i]
		num := fmt.Sprintf("%d", count-i)
		date := formatHumanDate(s.Time)
		age := formatRelativeAgo(s.Time, now)

		if !hasAPFS {
			ct.addRow(num, date, age)
			continue
		}

		var xid, delta, uuid, status string
		if s.UUID != "" {
			xid = fmt.Sprintf("%d", s.XID)
			uuid = s.UUID
			status = formatStatus(s)
			if i > 0 && snapshots[i-1].UUID != "" {
				delta = fmt.Sprintf("%d", s.XID-snapshots[i-1].XID)
			}
		}
		ct.addRow(num, date, age, xid, delta, uuid, status)
	}

	return ct.render(w, !noHeader)
}

func listTable(hasAPFS bool) *columnTable {
	cols := []columnDef{
		{title: "#", align: alignRight},
		{title: "DATE", align: alignLeft},
		{title: "AGE", align: alignLeft},
	}
	if hasAPFS {
		cols = append(cols,
			columnDef{title: "XID", align: alignRight},
			columnDef{title: "DELTA", align: alignRight},
			columnDef{title: "UUID", align: alignLeft},
			columnDef{title: "STATUS", align: alignLeft},
		)
	}
	return &columnTable{cols: cols}
}

func formatHumanDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func formatStatus(s snapshot.Snapshot) string {
	var parts []string
	if s.Purgeable {
		parts = append(parts, "purgeable")
	} else {
		parts = append(parts, "pinned")
	}
	if s.LimitsShrink {
		parts = append(parts, "limits shrink")
	}
	return strings.Join(parts, ", ")
}

func formatRelativeAgo(t, now time.Time) string {
	relative := snapshot.FormatRelativeTime(t, now)
	if relative == "future" {
		return relative
	}
	return relative + " ago"
}
