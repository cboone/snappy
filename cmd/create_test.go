package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
)

func TestCreateHumanOutput(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			output: []byte("Created local snapshot with date: 2026-03-01-143025\n"),
		},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	createCmd.SetOut(&buf)
	setFlag(t, createCmd, "json", "false")

	setTestContext(createCmd)
	if err := runCreate(createCmd, nil); err != nil {
		t.Fatalf("runCreate() error = %v", err)
	}

	want := "Snapshot created: 2026-03-01-143025\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestCreateJSONOutput(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			output: []byte("Created local snapshot with date: 2026-03-01-143025\n"),
		},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	createCmd.SetOut(&buf)
	setFlag(t, createCmd, "json", "true")
	defer setFlag(t, createCmd, "json", "false")

	setTestContext(createCmd)
	if err := runCreate(createCmd, nil); err != nil {
		t.Fatalf("runCreate() error = %v", err)
	}

	var result struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, buf.String())
	}
	if result.Date != "2026-03-01-143025" {
		t.Errorf("JSON date = %q, want %q", result.Date, "2026-03-01-143025")
	}
}

func TestCreateSnapshotError(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			err: fmt.Errorf("permission denied"),
		},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	createCmd.SetOut(&buf)
	setFlag(t, createCmd, "json", "false")

	setTestContext(createCmd)
	err := runCreate(createCmd, nil)
	if err == nil {
		t.Error("expected error from failed snapshot creation, got nil")
	}
}

func TestCreateCommandHasJSONFlag(t *testing.T) {
	f := createCmd.Flags().Lookup("json")
	if f == nil {
		t.Fatal("createCmd missing --json flag")
	}
	if f.DefValue != "false" {
		t.Errorf("--json default = %q, want %q", f.DefValue, "false")
	}
}
