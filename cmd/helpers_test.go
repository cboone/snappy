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

	cfg := config.Load()
	snapshots, apfsVolume, otherCount, err := loadSnapshots(context.Background(), runner, cfg)
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

	cfg := config.Load()
	_, _, _, err := loadSnapshots(context.Background(), runner, cfg)
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
