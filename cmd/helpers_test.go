package cmd

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/spf13/viper"

	"github.com/cboone/snappy/internal/config"
)

func TestLoadSnapshots(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {
			output: []byte("Snapshot dates for all disks:\n2026-03-01-140000\n2026-03-01-140100\n"),
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
	}}

	snapshots, apfsVolume, otherCount, err := loadSnapshots(context.Background(), runner)
	if err != nil {
		t.Fatalf("loadSnapshots() error = %v", err)
	}

	if len(snapshots) != 2 {
		t.Errorf("loadSnapshots() returned %d snapshots, want 2", len(snapshots))
	}
	if apfsVolume != "disk3s5" {
		t.Errorf("loadSnapshots() apfsVolume = %q, want %q", apfsVolume, "disk3s5")
	}
	if otherCount != 1 {
		t.Errorf("loadSnapshots() otherCount = %d, want 1", otherCount)
	}

	// Verify APFS details were merged.
	if snapshots[0].UUID != "ABC-123" {
		t.Errorf("snapshots[0].UUID = %q, want %q", snapshots[0].UUID, "ABC-123")
	}
}

func TestLoadSnapshotsListError(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	config.SetDefaults()

	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil listlocalsnapshotdates /": {
			err: fmt.Errorf("tmutil failed"),
		},
	}}

	_, _, _, err := loadSnapshots(context.Background(), runner)
	if err == nil {
		t.Error("loadSnapshots() expected error, got nil")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	err := writeJSON(&buf, struct {
		Name string `json:"name"`
	}{Name: "test"})
	if err != nil {
		t.Fatalf("writeJSON() error = %v", err)
	}

	want := "{\n  \"name\": \"test\"\n}\n"
	if buf.String() != want {
		t.Errorf("writeJSON() = %q, want %q", buf.String(), want)
	}
}

func TestDeleteSnapshotsSuccess(t *testing.T) {
	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil status": {output: []byte("Backup session status:\n{\n    Running = 0;\n}\n")},
		"tmutil deletelocalsnapshots 2026-03-01-140000": {output: []byte("Deleted\n")},
		"tmutil deletelocalsnapshots 2026-03-01-140100": {output: []byte("Deleted\n")},
	}}

	deleted, skipped, err := deleteSnapshots(context.Background(), runner, []string{"2026-03-01-140000", "2026-03-01-140100"})
	if err != nil {
		t.Fatalf("deleteSnapshots() error = %v", err)
	}
	if skipped != "" {
		t.Fatalf("unexpected skip: %s", skipped)
	}
	if deleted != 2 {
		t.Errorf("deleteSnapshots() deleted = %d, want 2", deleted)
	}
}

func TestDeleteSnapshotsPartialFailure(t *testing.T) {
	runner := &mockRunner{responses: map[string]mockResponse{
		"tmutil status": {output: []byte("Backup session status:\n{\n    Running = 0;\n}\n")},
		"tmutil deletelocalsnapshots 2026-03-01-140000": {output: []byte("Deleted\n")},
		"tmutil deletelocalsnapshots 2026-03-01-140100": {err: fmt.Errorf("permission denied")},
	}}

	deleted, skipped, err := deleteSnapshots(context.Background(), runner, []string{"2026-03-01-140000", "2026-03-01-140100"})
	if err == nil {
		t.Error("deleteSnapshots() expected error for partial failure, got nil")
	}
	if skipped != "" {
		t.Fatalf("unexpected skip: %s", skipped)
	}
	if deleted != 1 {
		t.Errorf("deleteSnapshots() deleted = %d, want 1", deleted)
	}
}

func testInfoPlist(device string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>DeviceIdentifier</key>
	<string>%s</string>
</dict>
</plist>`, device)
}

const testTMSnapshotsPlist = `<?xml version="1.0" encoding="UTF-8"?>
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
			<key>Purgeable</key>
			<true/>
			<key>LimitingContainerShrink</key>
			<false/>
		</dict>
		<dict>
			<key>SnapshotName</key>
			<string>com.apple.os.update-XYZ</string>
			<key>SnapshotUUID</key>
			<string>GHI-789</string>
			<key>Purgeable</key>
			<false/>
		</dict>
	</array>
</dict>
</plist>`

// backupRunner records every deletion and starts a backup after idleChecks probes.
type backupRunner struct {
	idleChecks int
	checks     int
	deleted    []string
	statusErr  error
}

func (r *backupRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	if name != "tmutil" {
		return nil, fmt.Errorf("unexpected command: %s", name)
	}
	switch args[0] {
	case "status":
		r.checks++
		if r.statusErr != nil {
			return nil, r.statusErr
		}
		if r.checks <= r.idleChecks {
			return []byte("Running = 0;"), nil
		}
		return []byte("Running = 1;"), nil
	case "deletelocalsnapshots":
		r.deleted = append(r.deleted, args[1])
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected tmutil command: %s", args[0])
	}
}

func TestDeleteSnapshotsSkipsBackup(t *testing.T) {
	for _, tt := range []struct {
		name       string
		idleChecks int
		statusErr  error
		want       int
	}{
		{name: "active backup"},
		{name: "backup starts during pass", idleChecks: 1, want: 1},
		{name: "status unavailable", statusErr: fmt.Errorf("unavailable")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &backupRunner{idleChecks: tt.idleChecks, statusErr: tt.statusErr}
			deleted, skipped, err := deleteSnapshots(context.Background(), r,
				[]string{"2026-11-17-114619", "2026-11-17-120115"})
			if err != nil || skipped == "" || deleted != tt.want || len(r.deleted) != tt.want {
				t.Fatalf("backup protection: deleted=%d commands=%v skipped=%q err=%v", deleted, r.deleted, skipped, err)
			}
			// The next pass must resume normally once the backup is idle.
			r.statusErr = nil
			r.idleChecks = r.checks + 2
			deleted, skipped, err = deleteSnapshots(context.Background(), r,
				[]string{"2026-11-17-114619", "2026-11-17-120115"})
			if err != nil || skipped != "" || deleted != 2 {
				t.Fatalf("idle retry: deleted=%d skipped=%q err=%v", deleted, skipped, err)
			}
		})
	}
}
