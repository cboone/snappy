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

	logLine(w, "STARTUP", "snappy run (interval=%s, thin >%s to %s)",
		cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence)

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run first iteration immediately.
	runIteration(ctx, w, runner, cfg)

	ticker := time.NewTicker(cfg.AutoSnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logLine(w, "SHUTDOWN", "signal received, exiting")
			return nil
		case <-ticker.C:
			runIteration(ctx, w, runner, cfg)
		}
	}
}

func runIteration(ctx context.Context, w io.Writer, runner platform.CommandRunner, cfg *config.Config) {
	// Create snapshot.
	createCtx, createCancel := context.WithTimeout(ctx, time.Minute)
	date, err := platform.CreateSnapshot(createCtx, runner)
	createCancel()

	if err != nil {
		logLine(w, "ERROR", "create snapshot: %v", err)
	} else if date == "" {
		logLine(w, "SNAPSHOT", "Created: <unknown date>")
	} else {
		logLine(w, "SNAPSHOT", "Created: %s", date)
	}

	// Load snapshots for thinning and count.
	loadCtx, loadCancel := context.WithTimeout(ctx, 30*time.Second)
	snapshots, _, _, err := loadSnapshots(loadCtx, runner, cfg)
	loadCancel()

	if err != nil {
		logLine(w, "ERROR", "list snapshots: %v", err)
		return
	}

	// Thin old snapshots.
	now := time.Now()
	mgr := snapshot.NewAutoManager(true, cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence, now)
	targets := mgr.ComputeThinTargets(snapshots, now)
	currentCount := len(snapshots)

	if len(targets) > 0 {
		deleted, deleteErr := deleteSnapshots(ctx, runner, targets)
		if deleteErr != nil {
			logLine(w, "ERROR", "thin: %v", deleteErr)
		}
		currentCount -= deleted
		if currentCount < 0 {
			currentCount = 0
		}
		logLine(w, "THIN", "Thinned %d snapshot(s)", deleted)
	}

	logLine(w, "LIST", "%d snapshot(s)", currentCount)
}

func logLine(w io.Writer, event, format string, args ...any) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(w, "[%s] %-8s %s\n", ts, event, msg)
}
