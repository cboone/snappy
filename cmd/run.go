package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/logger"
	"github.com/cboone/snappy/internal/platform"
	"github.com/cboone/snappy/internal/service"
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

	// Acquire exclusive lock to prevent concurrent auto-snapshot processes.
	lockPath := service.DefaultLockPath(cfg.LogDir)
	lock, err := service.Acquire(lockPath)
	if err != nil {
		if errors.Is(err, service.ErrLocked) {
			return fmt.Errorf("%w (lock: %s)", err, lockPath)
		}
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer func() { _ = lock.Release() }()

	// Create a shared logger that writes to the snappy.log file.
	log := logger.New(logger.Options{
		LogDir:     cfg.LogDir,
		MaxEntries: 0,
		MaxSize:    cfg.LogMaxSize,
		MaxFiles:   cfg.LogMaxFiles,
	})
	defer log.Close()

	runner := newRunner()
	w := cmd.OutOrStdout()

	if err := dualLog(w, log, logger.Startup, "snappy run (interval=%s, thin >%s to %s)",
		cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run first iteration immediately.
	if err := runIteration(ctx, w, log, runner, cfg); err != nil {
		return err
	}

	ticker := time.NewTicker(cfg.AutoSnapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return dualLog(w, log, logger.Info, "signal received, exiting")
		case <-ticker.C:
			if err := runIteration(ctx, w, log, runner, cfg); err != nil {
				return err
			}
		}
	}
}

func runIteration(ctx context.Context, w io.Writer, log *logger.Logger, runner platform.CommandRunner, cfg *config.Config) error {
	// Create snapshot.
	createCtx, createCancel := context.WithTimeout(ctx, time.Minute)
	date, err := platform.CreateSnapshot(createCtx, runner)
	createCancel()

	switch {
	case err != nil:
		if err := dualLog(w, log, logger.Error, "create snapshot: %v", err); err != nil {
			return err
		}
	case date == "":
		if err := dualLog(w, log, logger.Created, "Created: <unknown date>"); err != nil {
			return err
		}
	default:
		if err := dualLog(w, log, logger.Created, "Created: %s", date); err != nil {
			return err
		}
	}

	// Load snapshots for thinning and count.
	loadCtx, loadCancel := context.WithTimeout(ctx, 30*time.Second)
	snapshots, _, _, err := loadSnapshots(loadCtx, runner)
	loadCancel()

	if err != nil {
		return dualLog(w, log, logger.Error, "list snapshots: %v", err)
	}

	// Thin old snapshots.
	now := time.Now()
	mgr := snapshot.NewAutoManager(true, cfg.AutoSnapshotInterval, cfg.ThinAgeThreshold, cfg.ThinCadence, now)
	targets := mgr.ComputeThinTargets(snapshots, now, nil)
	currentCount := len(snapshots)

	if len(targets) > 0 {
		deleted, deleteErr := deleteSnapshots(ctx, runner, targets)
		if deleteErr != nil {
			if err := dualLog(w, log, logger.Error, "thin: %v", deleteErr); err != nil {
				return err
			}
		}
		currentCount -= deleted
		if currentCount < 0 {
			currentCount = 0
		}
		if err := dualLog(w, log, logger.Thinned, "Thinned %d snapshot(s)", deleted); err != nil {
			return err
		}
	}

	return dualLog(w, log, logger.Info, "%d snapshot(s)", currentCount)
}

// dualLog writes a log entry to both stdout (for terminal/launchd capture)
// and the shared logger (for the snappy.log file).
func dualLog(w io.Writer, log *logger.Logger, eventType logger.EventType, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	log.Log(eventType, msg)
	return logLine(w, string(eventType), "%s", msg)
}

// logLine writes a timestamped log line to the given writer.
func logLine(w io.Writer, event, format string, args ...any) error {
	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	_, err := fmt.Fprintf(w, "[%s] %-8s %s\n", ts, event, msg)
	return err
}
