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
	// Date should be in human-readable format.
	if !strings.Contains(output, "2026-03-01 14:01:00") {
		t.Errorf("output missing human-readable date, got: %s", output)
	}
	// Without APFS data, should show basic columns but no XID/UUID columns.
	if !strings.Contains(output, "DATE") {
		t.Errorf("output missing DATE header, got: %s", output)
	}
	if strings.Contains(output, "XID") {
		t.Errorf("output should not contain XID column without APFS data, got: %s", output)
	}
	if strings.Contains(output, "UUID") {
		t.Errorf("output should not contain UUID column without APFS data, got: %s", output)
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

const testAPFSSnapshotsWithXID = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Snapshots</key>
	<array>
		<dict>
			<key>SnapshotName</key>
			<string>com.apple.TimeMachine.2026-03-01-140000.local</string>
			<key>SnapshotUUID</key>
			<string>ABC-123</string>
			<key>SnapshotXID</key>
			<integer>1000</integer>
			<key>Purgeable</key>
			<true/>
			<key>LimitingContainerShrink</key>
			<false/>
		</dict>
		<dict>
			<key>SnapshotName</key>
			<string>com.apple.TimeMachine.2026-03-01-140100.local</string>
			<key>SnapshotUUID</key>
			<string>DEF-456</string>
			<key>SnapshotXID</key>
			<integer>1050</integer>
			<key>Purgeable</key>
			<false/>
			<key>LimitingContainerShrink</key>
			<true/>
		</dict>
	</array>
</dict>
</plist>`

func apfsListRunner() *mockRunner {
	return &mockRunner{responses: map[string]mockResponse{
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
			output: []byte(testAPFSSnapshotsWithXID),
		},
		"diskutil apfs listSnapshots disk3s1s1 -plist": {
			err: fmt.Errorf("unsupported"),
		},
	}}
}

func TestListJSONWithAPFSDetails(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }
	newRunner = func() platform.CommandRunner { return apfsListRunner() }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "true")
	defer setFlag(t, listCmd, "json", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	var result struct {
		Snapshots []struct {
			Date         string `json:"date"`
			UUID         string `json:"uuid"`
			XIDDelta     *int   `json:"xid_delta"`
			Purgeable    *bool  `json:"purgeable"`
			LimitsShrink *bool  `json:"limits_shrink"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if len(result.Snapshots) != 2 {
		t.Fatalf("snapshots = %d, want 2", len(result.Snapshots))
	}

	// First snapshot (oldest, index 0 in ascending order) has no xid_delta.
	first := result.Snapshots[0]
	if first.UUID != "ABC-123" {
		t.Errorf("first UUID = %q, want %q", first.UUID, "ABC-123")
	}
	if first.XIDDelta != nil {
		t.Errorf("first xid_delta = %v, want nil (no predecessor)", *first.XIDDelta)
	}
	if first.Purgeable == nil || !*first.Purgeable {
		t.Error("first purgeable should be true")
	}

	// Second snapshot (newer) has xid_delta = 50.
	second := result.Snapshots[1]
	if second.XIDDelta == nil {
		t.Fatal("second xid_delta should not be nil")
	}
	if *second.XIDDelta != 50 {
		t.Errorf("second xid_delta = %d, want 50", *second.XIDDelta)
	}
	if second.LimitsShrink == nil || !*second.LimitsShrink {
		t.Error("second limits_shrink should be true")
	}
	if second.Purgeable == nil || *second.Purgeable {
		t.Error("second purgeable should be false")
	}
}

func TestListJSONXIDDeltaOmittedForFirst(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }
	newRunner = func() platform.CommandRunner { return apfsListRunner() }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "true")
	defer setFlag(t, listCmd, "json", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	var result struct {
		Snapshots []struct {
			XIDDelta *int `json:"xid_delta"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if len(result.Snapshots) < 1 {
		t.Fatal("expected at least 1 snapshot")
	}
	if result.Snapshots[0].XIDDelta != nil {
		t.Errorf("first snapshot xid_delta = %v, want nil (no predecessor)", *result.Snapshots[0].XIDDelta)
	}
}

func TestListHumanWithAPFSDetails(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }
	newRunner = func() platform.CommandRunner { return apfsListRunner() }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	output := buf.String()
	// APFS columns should have headers.
	if !strings.Contains(output, "DELTA") {
		t.Errorf("output missing DELTA header, got:\n%s", output)
	}
	if !strings.Contains(output, "XID") {
		t.Errorf("output missing XID header, got:\n%s", output)
	}
	// Delta is now a standalone column value (not "delta:50").
	if !strings.Contains(output, "50") {
		t.Errorf("output missing delta value '50', got:\n%s", output)
	}
	if strings.Contains(output, "delta:") {
		t.Errorf("output should not contain 'delta:' prefix, got:\n%s", output)
	}
	// Status uses comma separator.
	if !strings.Contains(output, "pinned, limits shrink") {
		t.Errorf("output missing 'pinned, limits shrink', got:\n%s", output)
	}
	if !strings.Contains(output, "ABC-123") {
		t.Errorf("output missing UUID, got:\n%s", output)
	}
}

func TestListHumanNoHeader(t *testing.T) {
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
	setFlag(t, listCmd, "no-header", "true")
	defer setFlag(t, listCmd, "no-header", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	output := buf.String()
	// Summary line should still be present.
	if !strings.Contains(output, "2 snapshot(s)") {
		t.Errorf("output missing summary line, got: %s", output)
	}
	// Header row should be suppressed.
	if strings.Contains(output, "DATE") {
		t.Errorf("output should not contain header with --no-header, got: %s", output)
	}
	// Data rows should still be present.
	if !strings.Contains(output, "2026-03-01 14:01:00") {
		t.Errorf("output missing data row, got: %s", output)
	}
}

func TestListHumanNoHeaderWithAPFS(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }
	newRunner = func() platform.CommandRunner { return apfsListRunner() }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "false")
	setFlag(t, listCmd, "no-header", "true")
	defer setFlag(t, listCmd, "no-header", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	output := buf.String()
	// Header labels should be absent.
	if strings.Contains(output, "DELTA") {
		t.Errorf("output should not contain DELTA header, got:\n%s", output)
	}
	// Data should still be present.
	if !strings.Contains(output, "ABC-123") {
		t.Errorf("output missing UUID data, got:\n%s", output)
	}
	if !strings.Contains(output, "50") {
		t.Errorf("output missing delta value, got:\n%s", output)
	}
}

func TestListHumanNoHeaderEmpty(t *testing.T) {
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
	setFlag(t, listCmd, "no-header", "true")
	defer setFlag(t, listCmd, "no-header", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "0 snapshot(s)") {
		t.Errorf("output missing summary line, got: %s", output)
	}
}

func TestListHumanColumnAlignment(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	origNewRunner := newRunner
	origRequire := requireTmutil
	defer func() { newRunner = origNewRunner; requireTmutil = origRequire }()
	requireTmutil = func() error { return nil }
	newRunner = func() platform.CommandRunner { return apfsListRunner() }

	var buf bytes.Buffer
	listCmd.SetOut(&buf)
	setFlag(t, listCmd, "json", "false")
	setFlag(t, listCmd, "no-header", "false")

	setTestContext(listCmd)
	if err := runList(listCmd, nil); err != nil {
		t.Fatalf("runList() error = %v", err)
	}

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Find the header line (first line after the blank line).
	var headerIdx int
	for i, line := range lines {
		if strings.Contains(line, "DATE") && strings.Contains(line, "AGE") {
			headerIdx = i
			break
		}
	}
	if headerIdx == 0 {
		t.Fatalf("could not find header line in output:\n%s", output)
	}

	// DATE column should start at the same position in the header and data rows.
	headerDatePos := strings.Index(lines[headerIdx], "DATE")
	for i := headerIdx + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		// Each data row should have "2026-03-01" starting at the same column as "DATE".
		datePos := strings.Index(lines[i], "2026-03-01")
		if datePos != headerDatePos {
			t.Errorf("line %d: DATE column at position %d, want %d\nheader: %q\nrow:    %q",
				i, datePos, headerDatePos, lines[headerIdx], lines[i])
		}
	}
}
