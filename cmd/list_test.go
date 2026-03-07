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

func TestListHumanEmpty(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {output: []byte("")},
		"diskutil info -plist /":          {err: fmt.Errorf("no device")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	if !strings.Contains(buf.String(), "0 snapshot(s)") {
		t.Errorf("output missing '0 snapshot(s)', got: %s", buf.String())
	}
}

func TestListHumanWithSnapshots(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {
			output: []byte("2026-03-01-140000\n2026-03-01-140100\n"),
		},
		"diskutil info -plist /": {err: fmt.Errorf("no device")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2 snapshot(s)") {
		t.Errorf("output missing '2 snapshot(s)', got: %s", output)
	}
	if !strings.Contains(output, "2026-03-01-140100") {
		t.Errorf("output missing snapshot date, got: %s", output)
	}
}

func TestListJSONOutput(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {
			output: []byte("2026-03-01-140000\n2026-03-01-140100\n"),
		},
		"diskutil info -plist /": {err: fmt.Errorf("no device")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "true")
	defer setFlag(t, listCmd, "json", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	var result struct {
		Mount     string `json:"mount"`
		Count     int    `json:"count"`
		Snapshots []struct {
			Date     string `json:"date"`
			Relative string `json:"relative"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if result.Mount != "/" {
		t.Errorf("mount = %q, want %q", result.Mount, "/")
	}
	if result.Count != 2 {
		t.Errorf("count = %d, want 2", result.Count)
	}
	if len(result.Snapshots) != 2 {
		t.Errorf("snapshots length = %d, want 2", len(result.Snapshots))
	}
}

func TestListError(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {err: fmt.Errorf("tmutil failed")},
	}}
	newRunner = func() platform.CommandRunner { return runner }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "false")

	setTestContext(listCmd)
	err := runList(listCmd, nil)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestFormatRelativeAgoFuture(t *testing.T) {
	now := time.Now()
	future := now.Add(10 * time.Minute)
	got := formatRelativeAgo(future, now)
	if got != "future" {
		t.Errorf("formatRelativeAgo(future) = %q, want %q", got, "future")
	}
}
