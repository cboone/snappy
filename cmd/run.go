package cmd

import (
	"context"
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/snapshot"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the auto-snapshot loop (foreground daemon)",
	Args:  cobra.NoArgs,
	RunE:  runDaemon,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runDaemon(cmd *cobra.Command, _ []string) error {
	if err := requireTmutil(); err != nil {
		return err
	}

	cfg := config.Load()
	runner := newRunner()
	w := cmd.OutOrStdout()

	if err := logLine(w, "STARTUP", "snappy run (interval=%s, thin >%s to %s)",
		cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run first iteration immediately.
	if err := runIteration(ctx, w, runner, cfg); err != nil {
		return err
	}

	ticker := time.NewTicker(cfg.AutoSnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return logLine(w, "SHUTDOWN", "signal received, exiting")
		case <-ticker.C:
			if err := runIteration(ctx, w, runner, cfg); err != nil {
				return err
			}
		}
	}
}

func runIteration(ctx context.Context, w io.Writer, runner platform.CommandRunner, cfg *config.Config) error {
	// Create snapshot.
	createCtx, createCancel := context.WithTimeout(ctx, time.Minute)
	date, err := platform.CreateSnapshot(createCtx, runner)
	createCancel()

	switch {
	case err != nil:
		if err := logLine(w, "ERROR", "create snapshot: %v", err); err != nil {
			return err
		}
	case date == "":
		if err := logLine(w, "SNAPSHOT", "Created: <unknown date>"); err != nil {
			return err
		}
	default:
		if err := logLine(w, "SNAPSHOT", "Created: %s", date); err != nil {
			return err
		}
	}

	// Load snapshots for thinning and count.
	loadCtx, loadCancel := context.WithTimeout(ctx, 30*time.Second)
	snapshots, _, _, err := loadSnapshots(loadCtx, runner)
	loadCancel()

	if err != nil {
		return logLine(w, "ERROR", "list snapshots: %v", err)
	}

	// Thin old snapshots.
	now := time.Now()
	mgr := snapshot.NewAutoManager(true, cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence, now)
	targets := mgr.ComputeThinTargets(snapshots, now, nil)
	currentCount := len(snapshots)

	if len(targets) > 0 {
		deleted, deleteErr := deleteSnapshots(ctx, runner, targets)
		if deleteErr != nil {
			if err := logLine(w, "ERROR", "thin: %v", deleteErr); err != nil {
				return err
			}
		}
		currentCount -= deleted
		if currentCount < 0 {
			currentCount = 0
		}
		if err := logLine(w, "THIN", "Thinned %d snapshot(s)", deleted); err != nil {
			return err
		}
	}

	return logLine(w, "LIST", "%d snapshot(s)", currentCount)
}

func logLine(w io.Writer, event, format string, args ...any) error {
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	_, err := fmt.Fprintf(w, "[%s] %-8s %s\n", ts, event, msg)
	return err
}
