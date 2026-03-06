package platform

import (
	"context"
	"fmt"
	"testing"
)

// mockRunner returns canned output based on the command name and args.
type mockRunner struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	output []byte
	err    error
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}
	return nil, fmt.Errorf("unmocked command: %s", key)
}

func TestCheckStatus(t *testing.T) {
	tests := []struct {
		name   string
		output string
		err    error
		want   string
	}{
		{
			name:   "configured",
			output: "Name          : My Backup\nKind          : Network\n",
			want:   "Configured",
		},
		{
			name:   "not configured",
			output: "No destinations configured\n",
			want:   "Not configured (snapshots work regardless)",
		},
		{
			name: "error",
			err:  fmt.Errorf("exit status 1"),
			want: "Not configured (snapshots work regardless)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &mockRunner{responses: map[string]mockResponse{
				"tmutil destinationinfo": {output: []byte(tt.output), err: tt.err},
			}}
			got := CheckStatus(context.Background(), r)
			if got != tt.want {
				t.Errorf("CheckStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListSnapshots(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		err       error
		wantCount int
		wantErr   bool
	}{
		{
			name:      "two snapshots",
			output:    "Snapshot dates for all disks:\n2026-03-01-140000\n2026-03-01-140100\n",
			wantCount: 2,
		},
		{
			name:      "no snapshots",
			output:    "Snapshot dates for all disks:\n",
			wantCount: 0,
		},
		{
			name:    "error",
			err:     fmt.Errorf("exit status 1"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &mockRunner{responses: map[string]mockResponse{
				"tmutil listlocalsnapshotdates /": {output: []byte(tt.output), err: tt.err},
			}}
			dates, err := ListSnapshots(context.Background(), r, "/")
			if tt.wantErr {
				if err == nil {
					t.Error("ListSnapshots() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ListSnapshots() error = %v", err)
				return
			}
			if len(dates) != tt.wantCount {
				t.Errorf("ListSnapshots() returned %d dates, want %d", len(dates), tt.wantCount)
			}
		})
	}
}

func TestCreateSnapshot(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"tmutil localsnapshot": {
			output: []byte("Created local snapshot with date: 2026-03-01-143025\n"),
		},
	}}

	date, err := CreateSnapshot(context.Background(), r)
	if err != nil {
		t.Errorf("CreateSnapshot() error = %v", err)
	}
	if date != "2026-03-01-143025" {
		t.Errorf("CreateSnapshot() = %q, want %q", date, "2026-03-01-143025")
	}
}

func TestDeleteSnapshot(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"tmutil deletelocalsnapshots 2026-03-01-143025": {output: []byte("Deleted\n")},
	}}

	err := DeleteSnapshot(context.Background(), r, "2026-03-01-143025")
	if err != nil {
		t.Errorf("DeleteSnapshot() error = %v", err)
	}
}

func TestGetDiskInfo(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"df -h /": {
			output: []byte("Filesystem     Size   Used  Avail Capacity  iused ifree %iused  Mounted on\n/dev/disk3s1  460Gi  215Gi  242Gi    48%  3456789 2538923   58%   /\n"),
		},
	}}

	info, err := GetDiskInfo(context.Background(), r, "/")
	if err != nil {
		t.Errorf("GetDiskInfo() error = %v", err)
	}
	if info.Total != "460Gi" {
		t.Errorf("GetDiskInfo() total = %q, want %q", info.Total, "460Gi")
	}
	if info.Percent != "48%" {
		t.Errorf("GetDiskInfo() percent = %q, want %q", info.Percent, "48%")
	}
}

func TestGetSnapshotDetails(t *testing.T) {
	plistXML := `<?xml version="1.0" encoding="UTF-8"?>
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
			<string>YES</string>
			<key>LimitingContainerShrink</key>
			<string>YES</string>
		</dict>
		<dict>
			<key>SnapshotName</key>
			<string>com.apple.os.update-ABC123</string>
			<key>SnapshotUUID</key>
			<string>GHI-789</string>
			<key>Purgeable</key>
			<false/>
		</dict>
	</array>
</dict>
</plist>`

	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil apfs listSnapshots disk3s5 -plist": {output: []byte(plistXML)},
	}}

	details, otherCount, err := GetSnapshotDetails(context.Background(), r, "disk3s5")
	if err != nil {
		t.Fatalf("GetSnapshotDetails() error = %v", err)
	}

	if len(details) != 2 {
		t.Errorf("GetSnapshotDetails() returned %d details, want 2", len(details))
	}
	if otherCount != 1 {
		t.Errorf("GetSnapshotDetails() otherCount = %d, want 1", otherCount)
	}

	d1, ok := details["2026-03-01-140000"]
	if !ok {
		t.Fatal("missing detail for 2026-03-01-140000")
	}
	if d1.UUID != "ABC-123" {
		t.Errorf("UUID = %q, want %q", d1.UUID, "ABC-123")
	}
	if !d1.Purgeable {
		t.Error("expected Purgeable = true")
	}
	if d1.LimitsShrink {
		t.Error("expected LimitsShrink = false")
	}

	d2, ok := details["2026-03-01-140100"]
	if !ok {
		t.Fatal("missing detail for 2026-03-01-140100")
	}
	if !d2.Purgeable {
		t.Error("expected Purgeable = true (from string YES)")
	}
	if !d2.LimitsShrink {
		t.Error("expected LimitsShrink = true (from string YES)")
	}
}

func TestGetSnapshotDetailsEmptyVolume(t *testing.T) {
	details, otherCount, err := GetSnapshotDetails(context.Background(), nil, "")
	if err != nil {
		t.Errorf("GetSnapshotDetails('') error = %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details for empty volume, got %v", details)
	}
	if otherCount != 0 {
		t.Errorf("expected 0 other count for empty volume, got %d", otherCount)
	}
}

func TestParseBoolish(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string YES", "YES", true},
		{"string yes", "yes", true},
		{"string NO", "NO", false},
		{"string true", "true", true},
		{"string false", "false", false},
		{"nil", nil, false},
		{"int", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBoolish(tt.val)
			if got != tt.want {
				t.Errorf("parseBoolish(%v) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestFindAPFSVolumeRootFallsBackToDataVolume(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil info -plist /":                     {output: []byte(infoPlist("disk3s1s1"))},
		"diskutil info -plist /System/Volumes/Data":  {output: []byte(infoPlist("disk3s5"))},
		"diskutil apfs listSnapshots disk3s5 -plist": {output: []byte(tmSnapshotsPlist)},
		"diskutil apfs listSnapshots disk3s1s1 -plist": {
			err: fmt.Errorf("listSnapshots not supported for sealed snapshot mount"),
		},
	}}

	got, err := FindAPFSVolume(context.Background(), r, "/")
	if err != nil {
		t.Fatalf("FindAPFSVolume() error = %v", err)
	}
	if got != "disk3s5" {
		t.Fatalf("FindAPFSVolume() = %q, want %q", got, "disk3s5")
	}
}

func TestFindAPFSVolumeSkipsVolumeWithNoTMSnapshots(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil info -plist /Volumes/External":     {output: []byte(infoPlist("disk4s1"))},
		"diskutil apfs listSnapshots disk4s1 -plist": {output: []byte(emptySnapshotsPlist)},
	}}

	got, err := FindAPFSVolume(context.Background(), r, "/Volumes/External")
	if err != nil {
		t.Fatalf("FindAPFSVolume() error = %v", err)
	}
	if got != "" {
		t.Fatalf("FindAPFSVolume() = %q, want empty (no TM snapshots)", got)
	}
}

func TestFindAPFSVolumeSelectsVolumeWithTMSnapshots(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil info -plist /Volumes/External":     {output: []byte(infoPlist("disk4s1"))},
		"diskutil apfs listSnapshots disk4s1 -plist": {output: []byte(tmSnapshotsPlist)},
	}}

	got, err := FindAPFSVolume(context.Background(), r, "/Volumes/External")
	if err != nil {
		t.Fatalf("FindAPFSVolume() error = %v", err)
	}
	if got != "disk4s1" {
		t.Fatalf("FindAPFSVolume() = %q, want %q", got, "disk4s1")
	}
}

func TestFindAPFSVolumeReturnsEmptyWhenNoDeviceSupportsSnapshots(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil info -plist /":                       {output: []byte(infoPlist("disk3s1s1"))},
		"diskutil info -plist /System/Volumes/Data":    {output: []byte(infoPlist("disk3s5"))},
		"diskutil apfs listSnapshots disk3s5 -plist":   {err: fmt.Errorf("unsupported")},
		"diskutil apfs listSnapshots disk3s1s1 -plist": {err: fmt.Errorf("unsupported")},
	}}

	got, err := FindAPFSVolume(context.Background(), r, "/")
	if err != nil {
		t.Fatalf("FindAPFSVolume() error = %v", err)
	}
	if got != "" {
		t.Fatalf("FindAPFSVolume() = %q, want empty", got)
	}
}

func TestGetMountInfo(t *testing.T) {
	plistXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>DeviceIdentifier</key>
	<string>disk3s1s1</string>
	<key>VolumeName</key>
	<string>Macintosh HD</string>
	<key>APFSContainerReference</key>
	<string>disk3</string>
</dict>
</plist>`

	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil info -plist /": {output: []byte(plistXML)},
	}}

	info, err := GetMountInfo(context.Background(), r, "/")
	if err != nil {
		t.Fatalf("GetMountInfo() error = %v", err)
	}
	if info.DeviceIdentifier != "disk3s1s1" {
		t.Errorf("DeviceIdentifier = %q, want %q", info.DeviceIdentifier, "disk3s1s1")
	}
	if info.VolumeName != "Macintosh HD" {
		t.Errorf("VolumeName = %q, want %q", info.VolumeName, "Macintosh HD")
	}
	if info.APFSContainerReference != "disk3" {
		t.Errorf("APFSContainerReference = %q, want %q", info.APFSContainerReference, "disk3")
	}
}

func TestGetContainerReference(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil info -plist /": {output: []byte(infoPlistWithContainer("disk3s1s1", "disk3"))},
	}}

	got, err := GetContainerReference(context.Background(), r, "/")
	if err != nil {
		t.Fatalf("GetContainerReference() error = %v", err)
	}
	if got != "disk3" {
		t.Errorf("GetContainerReference() = %q, want %q", got, "disk3")
	}
}

func TestGetContainerReferenceEmpty(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil info -plist /": {output: []byte(infoPlist("disk3s1s1"))},
	}}

	got, err := GetContainerReference(context.Background(), r, "/")
	if err != nil {
		t.Fatalf("GetContainerReference() error = %v", err)
	}
	if got != "" {
		t.Errorf("GetContainerReference() = %q, want empty", got)
	}
}

func TestGetContainerTidemark(t *testing.T) {
	plistXML := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>MinimumSizeNoGuard</key>
	<integer>2153406005248</integer>
	<key>ContainerCurrentSize</key>
	<integer>4000787030016</integer>
</dict>
</plist>`

	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil apfs resizeContainer disk3 limits -plist": {output: []byte(plistXML)},
	}}

	got, err := GetContainerTidemark(context.Background(), r, "disk3")
	if err != nil {
		t.Fatalf("GetContainerTidemark() error = %v", err)
	}
	if got != 2153406005248 {
		t.Errorf("GetContainerTidemark() = %d, want %d", got, 2153406005248)
	}
}

func TestGetContainerTidemarkError(t *testing.T) {
	r := &mockRunner{responses: map[string]mockResponse{
		"diskutil apfs resizeContainer disk3 limits -plist": {err: fmt.Errorf("not supported")},
	}}

	_, err := GetContainerTidemark(context.Background(), r, "disk3")
	if err == nil {
		t.Error("GetContainerTidemark() expected error, got nil")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		b    int64
		want string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 104857600, "100.0 MB"},
		{"gigabytes", 494384128819, "460.4 GB"},
		{"terabytes", 2153406005248, "2.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.b)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.b, got, tt.want)
			}
		})
	}
}

func infoPlist(device string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>DeviceIdentifier</key>
	<string>%s</string>
</dict>
</plist>`, device)
}

func infoPlistWithContainer(device, container string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>DeviceIdentifier</key>
	<string>%s</string>
	<key>APFSContainerReference</key>
	<string>%s</string>
</dict>
</plist>`, device, container)
}

const emptySnapshotsPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Snapshots</key>
	<array/>
</dict>
</plist>`

const tmSnapshotsPlist = `<?xml version="1.0" encoding="UTF-8"?>
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
		</dict>
	</array>
</dict>
</plist>`
