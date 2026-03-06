package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/logger"
)

func TestRunIterationLogsAndContinues(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	// Create a runner where snapshot creation fails but listing succeeds.
	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			err: fmt.Errorf("permission denied"),
		},
		"tmutil listlocalsnapshotdates /": {
			output: []byte("2026-03-01-140000\n"),
		},
		"diskutil info -plist /": {err: fmt.Errorf("no device")},
	}}

	cfg := config.Load()
	log := logger.New(logger.Options{})
	defer log.Close()
	var buf bytes.Buffer

	// runIteration should not panic or return an error; it logs and continues.
	if err := runIteration(context.Background(), &buf, log, runner, cfg); err != nil {
		t.Fatalf("runIteration returned unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("output missing ERROR log line, got:\n%s", output)
	}
	if !strings.Contains(output, "permission denied") {
		t.Errorf("output missing error details, got:\n%s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("output missing INFO log line (should continue after error), got:\n%s", output)
	}
}

func TestRunIterationSuccess(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	now := time.Now()
	date := now.Format("2006-01-02-150405")

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			output: []byte("Created local snapshot with date: " + date + "\n"),
		},
		"tmutil listlocalsnapshotdates /": {
			output: []byte(date + "\n"),
		},
		"diskutil info -plist /": {err: fmt.Errorf("no device")},
	}}

	cfg := config.Load()
	log := logger.New(logger.Options{})
	defer log.Close()
	var buf bytes.Buffer

	if err := runIteration(context.Background(), &buf, log, runner, cfg); err != nil {
		t.Fatalf("runIteration returned unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CREATED") {
		t.Errorf("output missing SNAPSHOT log line, got:\n%s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("output missing INFO log line, got:\n%s", output)
	}
	if strings.Contains(output, "ERROR") {
		t.Errorf("unexpected ERROR in output:\n%s", output)
	}
}

func TestRunIterationLogsPostThinCount(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()
	viper.Set("thin_age_threshold", "1s")
	viper.Set("thin_cadence", "1h")

	now := time.Now()
	oldest := now.Add(-3 * time.Hour).Format("2006-01-02-150405")
	middle := now.Add(-150 * time.Minute).Format("2006-01-02-150405")

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			err: fmt.Errorf("permission denied"),
		},
		"tmutil listlocalsnapshotdates /": {
			output: []byte(oldest + "\n" + middle + "\n"),
		},
		"tmutil deletelocalsnapshots " + middle: {},
		"diskutil info -plist /":                {err: fmt.Errorf("no device")},
	}}

	cfg := config.Load()
	log := logger.New(logger.Options{})
	defer log.Close()
	var buf bytes.Buffer

	if err := runIteration(context.Background(), &buf, log, runner, cfg); err != nil {
		t.Fatalf("runIteration returned unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "THINNED") {
		t.Fatalf("output missing THIN log line, got:\n%s", output)
	}
	if !strings.Contains(output, "Thinned 1 snapshot(s)") {
		t.Fatalf("output missing thin count, got:\n%s", output)
	}
	if !strings.Contains(output, "INFO") || !strings.Contains(output, "1 snapshot(s)") {
		t.Fatalf("output missing post-thin list count, got:\n%s", output)
	}
}

func TestRunIterationContextCancelled(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	// Use a cancelled context to verify iteration handles cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			err: fmt.Errorf("context canceled"),
		},
		"tmutil listlocalsnapshotdates /": {
			err: fmt.Errorf("context canceled"),
		},
	}}

	cfg := config.Load()
	log := logger.New(logger.Options{})
	defer log.Close()
	var buf bytes.Buffer

	// Should not panic even with cancelled context.
	if err := runIteration(ctx, &buf, log, runner, cfg); err != nil {
		t.Fatalf("runIteration returned unexpected error: %v", err)
	}
}

func TestRunCommandStructure(t *testing.T) {
	if runCmd.Use != "run" {
		t.Errorf("runCmd.Use = %q, want %q", runCmd.Use, "run")
	}

	// Verify run does NOT have a --json flag.
	f := runCmd.Flags().Lookup("json")
	if f != nil {
		t.Error("runCmd should not have --json flag")
	}
}

func TestLogLine(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var buf bytes.Buffer
		if err := logLine(&buf, "TEST", "message %d", 42); err != nil {
			t.Fatalf("logLine returned unexpected error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "TEST") {
			t.Errorf("logLine output missing event, got: %s", output)
		}
		if !strings.Contains(output, "message 42") {
			t.Errorf("logLine output missing message, got: %s", output)
		}
		// Verify timestamp format [YYYY-MM-DD HH:MM:SS].
		if output[0] != '[' {
			t.Errorf("logLine output should start with '[', got: %s", output)
		}
	})

	t.Run("write error", func(t *testing.T) {
		w := &errWriter{err: fmt.Errorf("broken pipe")}
		if err := logLine(w, "TEST", "message"); err == nil {
			t.Fatal("logLine should return error on write failure")
		}
	})
}

func TestDualLogWritesToBothDestinations(t *testing.T) {
	log := logger.New(logger.Options{MaxEntries: 50})
	defer log.Close()
	var buf bytes.Buffer

	if err := dualLog(&buf, log, logger.Info, "test message %d", 99); err != nil {
		t.Fatalf("dualLog returned unexpected error: %v", err)
	}

	// Verify writer output.
	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("writer output missing event type, got: %s", output)
	}
	if !strings.Contains(output, "test message 99") {
		t.Errorf("writer output missing message, got: %s", output)
	}

	// Verify logger received the entry.
	entries := log.Entries()
	if len(entries) == 0 {
		t.Fatal("logger has no entries after dualLog")
	}
	last := entries[len(entries)-1]
	if last.Type != logger.Info {
		t.Errorf("logger entry type = %q, want %q", last.Type, logger.Info)
	}
	if !strings.Contains(last.Message, "test message 99") {
		t.Errorf("logger entry message = %q, want to contain %q", last.Message, "test message 99")
	}
}

// errWriter is an io.Writer that always returns an error.
type errWriter struct {
	err error
}

func (w *errWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}
