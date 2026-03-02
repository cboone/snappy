package platform

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	snapshotDateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-\d{6}$`)
	snapshotDateExtRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}-\d{6}`)
)

// CheckStatus runs tmutil destinationinfo and returns a status string.
func CheckStatus(ctx context.Context, r CommandRunner) string {
	out, err := r.Run(ctx, "tmutil", "destinationinfo")
	if err != nil {
		return "Not configured (snapshots work regardless)"
	}
	if strings.Contains(string(out), "No destinations configured") {
		return "Not configured (snapshots work regardless)"
	}
	return "Configured"
}

// ListSnapshots returns sorted snapshot date strings from tmutil.
func ListSnapshots(ctx context.Context, r CommandRunner, mount string) ([]string, error) {
	out, err := r.Run(ctx, "tmutil", "listlocalsnapshotdates", mount)
	if err != nil {
		return nil, fmt.Errorf("listing snapshots: %w", err)
	}

	var dates []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if snapshotDateRe.MatchString(line) {
			dates = append(dates, line)
		}
	}
	sort.Strings(dates)
	return dates, nil
}

// CreateSnapshot creates a new local snapshot and returns the date string
// extracted from tmutil's output.
func CreateSnapshot(ctx context.Context, r CommandRunner) (string, error) {
	out, err := r.Run(ctx, "tmutil", "localsnapshot")
	if err != nil {
		return "", fmt.Errorf("creating snapshot: %w", err)
	}

	// Extract date from output like "Created local snapshot with date: 2026-03-01-143025"
	if match := snapshotDateExtRe.FindString(string(out)); match != "" {
		return match, nil
	}
	return "", nil
}

// DeleteSnapshot deletes a local snapshot by date string.
func DeleteSnapshot(ctx context.Context, r CommandRunner, date string) error {
	_, err := r.Run(ctx, "tmutil", "deletelocalsnapshots", date)
	if err != nil {
		return fmt.Errorf("deleting snapshot %s: %w", date, err)
	}
	return nil
}
