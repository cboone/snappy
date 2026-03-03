package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
	"github.com/cboone/snappy/internal/platform"
)

func TestStatusHumanOutput(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil destinationinfo": {
			output: []byte("Name          : My Backup\n"),
		},
		"tmutil listlocalsnapshotdates /": {
			output: []byte("2026-03-01-140000\n2026-03-01-140100\n"),
		},
		"diskutil info -plist /": {
			output: []byte(testInfoPlist("disk3s1s1")),
		},
		"diskutil info -plist /System/Volumes/Data": {
			output: []byte(testInfoPlist("disk3s5")),
		},
		"diskutil apfs listSnapshots disk3s5 -plist": {
			output: []byte(testTMSnapshotsPlist),
		},
		"diskutil apfs listSnapshots disk3s1s1 -plist": {
			err: fmt.Errorf("unsupported"),
		},
		"df -h /": {
			output: []byte("Filesystem     Size   Used  Avail Capacity\n/dev/disk3s1  466Gi  280Gi  186Gi    60%\n"),
		},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	statusCmd.SetOut(&buf)
	setFlag(t, statusCmd, "json", "false")

	if err := runStatus(statusCmd, nil); err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}

	output := buf.String()
	checks := []string{
		"Time Machine: Configured",
		"Mount: /",
		"APFS volume: disk3s5",
		"Disk:",
		"Snapshots: 2 local",
		"Auto-snapshot: enabled",
	}
	for _, c := range checks {
		if !strings.Contains(output, c) {
			t.Errorf("output missing %q, got:\n%s", c, output)
		}
	}
}

func TestStatusJSONOutput(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil destinationinfo": {
			output: []byte("Name          : My Backup\n"),
		},
		"tmutil listlocalsnapshotdates /": {
			output: []byte("2026-03-01-140000\n"),
		},
		"diskutil info -plist /": {
			err: fmt.Errorf("no device"),
		},
		"df -h /": {
			output: []byte("Filesystem     Size   Used  Avail Capacity\n/dev/disk3s1  466Gi  280Gi  186Gi    60%\n"),
		},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	statusCmd.SetOut(&buf)
	setFlag(t, statusCmd, "json", "true")
	defer setFlag(t, statusCmd, "json", "false")

	if err := runStatus(statusCmd, nil); err != nil {
		t.Fatalf("runStatus() error = %v", err)
	}

	var result struct {
		TimeMachine string `json:"time_machine"`
		Mount       string `json:"mount"`
		Snapshots   struct {
			Local int `json:"local"`
		} `json:"snapshots"`
		Auto struct {
			Enabled bool `json:"enabled"`
		} `json:"auto"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if result.TimeMachine != "Configured" {
		t.Errorf("time_machine = %q, want %q", result.TimeMachine, "Configured")
	}
	if result.Snapshots.Local != 1 {
		t.Errorf("snapshots.local = %d, want 1", result.Snapshots.Local)
	}
	if !result.Auto.Enabled {
		t.Error("auto.enabled = false, want true")
	}
}
