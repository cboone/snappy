package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
)

func TestThinNoTargets(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	// Snapshots too recent to thin (default thin_age_threshold is 10m).
	now := time.Now()
	date := now.Add(-1 * time.Minute).Format("2006-01-02-150405")

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {output: []byte(date + "\n")},
		"diskutil info -plist /":          {err: fmt.Errorf("no device")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	thinCmd.SetOut(&buf)
	setFlag(t, thinCmd, "json", "false")

	if err := runThin(thinCmd, nil); err != nil {
		t.Fatalf("runThin() error = %v", err)
	}

	if !strings.Contains(buf.String(), "No snapshots to thin") {
		t.Errorf("output = %q, want 'No snapshots to thin'", buf.String())
	}
}

func TestThinWithTargets(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	// Create three snapshots older than thin_age_threshold (10m) with gaps
	// smaller than thin_cadence (5m), so the middle one gets thinned.
	now := time.Now()
	d1 := now.Add(-20 * time.Minute).Format("2006-01-02-150405")
	d2 := now.Add(-19 * time.Minute).Format("2006-01-02-150405") // thin target
	d3 := now.Add(-12 * time.Minute).Format("2006-01-02-150405")

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {
			output: []byte(d1 + "\n" + d2 + "\n" + d3 + "\n"),
		},
		"diskutil info -plist /":            {err: fmt.Errorf("no device")},
		"tmutil deletelocalsnapshots " + d2: {output: []byte("Deleted\n")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	thinCmd.SetOut(&buf)
	setFlag(t, thinCmd, "json", "false")

	if err := runThin(thinCmd, nil); err != nil {
		t.Fatalf("runThin() error = %v", err)
	}

	if !strings.Contains(buf.String(), "Thinned 1 snapshot(s)") {
		t.Errorf("output = %q, want 'Thinned 1 snapshot(s)'", buf.String())
	}
}

func TestThinPartialFailure(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	now := time.Now()
	d1 := now.Add(-20 * time.Minute).Format("2006-01-02-150405")
	d2 := now.Add(-19 * time.Minute).Format("2006-01-02-150405") // thin target (will fail)
	d3 := now.Add(-12 * time.Minute).Format("2006-01-02-150405")

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {
			output: []byte(d1 + "\n" + d2 + "\n" + d3 + "\n"),
		},
		"diskutil info -plist /":            {err: fmt.Errorf("no device")},
		"tmutil deletelocalsnapshots " + d2: {err: fmt.Errorf("permission denied")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	thinCmd.SetOut(&buf)
	setFlag(t, thinCmd, "json", "false")

	err := runThin(thinCmd, nil)
	if err == nil {
		t.Error("expected error for partial failure, got nil")
	}

	// Should still report count before error.
	if !strings.Contains(buf.String(), "Thinned 0 snapshot(s)") {
		t.Errorf("output = %q, want 'Thinned 0 snapshot(s)'", buf.String())
	}
}

func TestThinJSONOutput(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	now := time.Now()
	date := now.Add(-1 * time.Minute).Format("2006-01-02-150405")

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {output: []byte(date + "\n")},
		"diskutil info -plist /":          {err: fmt.Errorf("no device")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	thinCmd.SetOut(&buf)
	setFlag(t, thinCmd, "json", "true")
	defer setFlag(t, thinCmd, "json", "false")

	if err := runThin(thinCmd, nil); err != nil {
		t.Fatalf("runThin() error = %v", err)
	}

	var result struct {
		Thinned int `json:"thinned"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if result.Thinned != 0 {
		t.Errorf("thinned = %d, want 0", result.Thinned)
	}
}
