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
	var buf bytes.Buffer

	// runIteration should not panic or return an error; it logs and continues.
	runIteration(context.Background(), &buf, runner, cfg)

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("output missing ERROR log line, got:\n%s", output)
	}
	if !strings.Contains(output, "permission denied") {
		t.Errorf("output missing error details, got:\n%s", output)
	}
	if !strings.Contains(output, "LIST") {
		t.Errorf("output missing LIST log line (should continue after error), got:\n%s", output)
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
	var buf bytes.Buffer

	runIteration(context.Background(), &buf, runner, cfg)

	output := buf.String()
	if !strings.Contains(output, "SNAPSHOT") {
		t.Errorf("output missing SNAPSHOT log line, got:\n%s", output)
	}
	if !strings.Contains(output, "LIST") {
		t.Errorf("output missing LIST log line, got:\n%s", output)
	}
	if strings.Contains(output, "ERROR") {
		t.Errorf("unexpected ERROR in output:\n%s", output)
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
	var buf bytes.Buffer

	// Should not panic even with cancelled context.
	runIteration(ctx, &buf, runner, cfg)
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
	var buf bytes.Buffer
	logLine(&buf, "TEST", "message %d", 42)

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
}
