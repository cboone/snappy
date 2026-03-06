package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

// requireTmutil checks that tmutil is available on the system.
// Tests can override this variable to skip the check.
var requireTmutil = func() error {
	if _, err := exec.LookPath("tmutil"); err != nil {
		return fmt.Errorf("tmutil not found: this tool requires macOS with Time Machine support")
	}
	return nil
}

// newRunner is a factory function for creating a CommandRunner.
// It returns a real OSRunner by default; tests override this variable.
var newRunner = func() platform.CommandRunner {
	return platform.OSRunner{}
}

// loadSnapshots fetches snapshots from tmutil, parses dates, discovers the APFS
// volume, and merges APFS details into each snapshot. Returns snapshots in
// ascending order (oldest first), the APFS volume identifier, the count of
// non-Time-Machine snapshots, and any error.
func loadSnapshots(ctx context.Context, runner platform.CommandRunner) (snapshots []snapshot.Snapshot, apfsVolume string, otherCount int, _ error) {
	dates, err := platform.ListSnapshots(ctx, runner, config.DefaultMount)
	if err != nil {
		return nil, "", 0, err
	}

	for _, d := range dates {
		t, parseErr := snapshot.ParseDate(d)
		if parseErr != nil {
			continue
		}
		snapshots = append(snapshots, snapshot.Snapshot{Date: d, Time: t})
	}

	apfsVolume, findErr := platform.FindAPFSVolume(ctx, runner, config.DefaultMount)
	if findErr != nil || apfsVolume == "" {
		return snapshots, "", 0, nil //nolint:nilerr // graceful degradation
	}

	details, otherCount, detailErr := platform.GetSnapshotDetails(ctx, runner, apfsVolume)
	if detailErr != nil {
		return snapshots, apfsVolume, 0, nil //nolint:nilerr // graceful degradation
	}

	for i, s := range snapshots {
		if d, ok := details[s.Date]; ok {
			snapshots[i].UUID = d.UUID
			snapshots[i].Purgeable = d.Purgeable
			snapshots[i].LimitsShrink = d.LimitsShrink
		}
	}

	return snapshots, apfsVolume, otherCount, nil
}

// writeJSON marshals v as indented JSON and writes it to w.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// deleteSnapshots deletes each snapshot date with an individual timeout,
// collecting failures. It returns the count of successful deletions and an
// error summarizing any failures.
func deleteSnapshots(ctx context.Context, runner platform.CommandRunner, targets []string) (int, error) {
	deleted := 0
	var failed []string

	for _, date := range targets {
		delCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := platform.DeleteSnapshot(delCtx, runner, date)
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
