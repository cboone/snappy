package platform

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"howett.net/plist"
)

// SnapshotDetail holds APFS metadata for a single snapshot.
type SnapshotDetail struct {
	Date         string
	UUID         string
	XID          int
	Purgeable    bool
	LimitsShrink bool
}

// APFSInfo holds the result of querying APFS snapshot details.
type APFSInfo struct {
	Volume         string
	Details        map[string]SnapshotDetail
	OtherSnapCount int
}

// diskutilInfoPlist is the subset of diskutil info -plist we need.
type diskutilInfoPlist struct {
	DeviceIdentifier       string `plist:"DeviceIdentifier"`
	VolumeName             string `plist:"VolumeName"`
	APFSContainerReference string `plist:"APFSContainerReference"`
}

// containerLimitsPlist holds the resize limits for an APFS container.
type containerLimitsPlist struct {
	MinimumSizeNoGuard   int64 `plist:"MinimumSizeNoGuard"`
	ContainerCurrentSize int64 `plist:"ContainerCurrentSize"`
}

// apfsSnapshotEntry represents a single snapshot in the APFS plist.
type apfsSnapshotEntry struct {
	SnapshotName            string      `plist:"SnapshotName"`
	SnapshotUUID            string      `plist:"SnapshotUUID"`
	SnapshotXID             int         `plist:"SnapshotXID"`
	Purgeable               interface{} `plist:"Purgeable"`
	LimitingContainerShrink interface{} `plist:"LimitingContainerShrink"`
}

// apfsSnapshotsPlist is the top-level plist from diskutil apfs listSnapshots.
type apfsSnapshotsPlist struct {
	Snapshots []apfsSnapshotEntry `plist:"Snapshots"`
}

var tmNameRe = regexp.MustCompile(`com\.apple\.TimeMachine\.(\d{4}-\d{2}-\d{2}-\d{6})\.local`)

// FindAPFSVolume discovers the APFS volume that holds Time Machine snapshots
// for the given mount point.
func FindAPFSVolume(ctx context.Context, r CommandRunner, mount string) (string, error) {
	device, err := getDeviceIdentifier(ctx, r, mount)
	if err != nil || device == "" {
		return "", err
	}

	// Root is mounted from a sealed system snapshot. Time Machine snapshots
	// live on the Data volume at /System/Volumes/Data instead.
	if mount == "/" {
		dataDevice, err := getDeviceIdentifier(ctx, r, "/System/Volumes/Data")
		if err == nil && dataDevice != "" && hasTimeMachineSnapshots(ctx, r, dataDevice) {
			return dataDevice, nil
		}
	}

	if hasTimeMachineSnapshots(ctx, r, device) {
		return device, nil
	}

	return "", nil
}

// GetSnapshotDetails retrieves APFS snapshot metadata for the given volume,
// correlating Time Machine snapshots by name.
func GetSnapshotDetails(ctx context.Context, r CommandRunner, volume string) (details map[string]SnapshotDetail, otherCount int, err error) {
	if volume == "" {
		return nil, 0, nil
	}

	out, err := r.Run(ctx, "diskutil", "apfs", "listSnapshots", volume, "-plist")
	if err != nil {
		return nil, 0, fmt.Errorf("listing APFS snapshots: %w", err)
	}

	var pl apfsSnapshotsPlist
	if _, err := plist.Unmarshal(out, &pl); err != nil {
		return nil, 0, fmt.Errorf("parsing APFS plist: %w", err)
	}

	details = make(map[string]SnapshotDetail)
	otherCount = 0

	for _, snap := range pl.Snapshots {
		match := tmNameRe.FindStringSubmatch(snap.SnapshotName)
		if match == nil {
			otherCount++
			continue
		}

		date := match[1]
		details[date] = SnapshotDetail{
			Date:         date,
			UUID:         snap.SnapshotUUID,
			XID:          snap.SnapshotXID,
			Purgeable:    parseBoolish(snap.Purgeable),
			LimitsShrink: parseBoolish(snap.LimitingContainerShrink),
		}
	}

	return details, otherCount, nil
}

// MountInfo holds the diskutil info fields for a given mount point.
type MountInfo struct {
	DeviceIdentifier       string
	VolumeName             string
	APFSContainerReference string
}

// GetMountInfo returns device identifier, volume name, and APFS container
// reference for the given mount point in a single diskutil info call.
func GetMountInfo(ctx context.Context, r CommandRunner, mount string) (MountInfo, error) {
	info, err := getDiskutilInfo(ctx, r, mount)
	if err != nil {
		return MountInfo{}, err
	}
	return MountInfo(info), nil
}

// GetVolumeName returns the human-friendly volume name for the given mount
// point by parsing the VolumeName field from diskutil info -plist.
func GetVolumeName(ctx context.Context, r CommandRunner, mount string) (string, error) {
	info, err := getDiskutilInfo(ctx, r, mount)
	if err != nil {
		return "", err
	}
	return info.VolumeName, nil
}

// GetContainerReference returns the APFS container reference (e.g. "disk3")
// for the given mount point by parsing diskutil info -plist.
func GetContainerReference(ctx context.Context, r CommandRunner, mount string) (string, error) {
	info, err := getDiskutilInfo(ctx, r, mount)
	if err != nil {
		return "", err
	}
	return info.APFSContainerReference, nil
}

// GetContainerTidemark returns the minimum container size in bytes
// (MinimumSizeNoGuard) for the given APFS container. This represents the
// lowest size the container can be resized to, constrained by file and
// snapshot usage.
func GetContainerTidemark(ctx context.Context, r CommandRunner, container string) (int64, error) {
	out, err := r.Run(ctx, "diskutil", "apfs", "resizeContainer", container, "limits", "-plist")
	if err != nil {
		return 0, fmt.Errorf("getting container limits for %s: %w", container, err)
	}

	var limits containerLimitsPlist
	if _, err := plist.Unmarshal(out, &limits); err != nil {
		return 0, fmt.Errorf("parsing container limits plist for %s: %w", container, err)
	}

	return limits.MinimumSizeNoGuard, nil
}

// getDiskutilInfo runs "diskutil info -plist" for the given mount point and
// returns the parsed result.
func getDiskutilInfo(ctx context.Context, r CommandRunner, mount string) (diskutilInfoPlist, error) {
	out, err := r.Run(ctx, "diskutil", "info", "-plist", mount)
	if err != nil {
		return diskutilInfoPlist{}, fmt.Errorf("getting diskutil info for %s: %w", mount, err)
	}

	var info diskutilInfoPlist
	if _, err := plist.Unmarshal(out, &info); err != nil {
		return diskutilInfoPlist{}, fmt.Errorf("parsing diskutil plist for %s: %w", mount, err)
	}

	return info, nil
}

func getDeviceIdentifier(ctx context.Context, r CommandRunner, mount string) (string, error) {
	info, err := getDiskutilInfo(ctx, r, mount)
	if err != nil {
		return "", err
	}
	return info.DeviceIdentifier, nil
}

// hasTimeMachineSnapshots checks whether the given volume contains at least
// one Time Machine snapshot by parsing the APFS snapshot listing.
func hasTimeMachineSnapshots(ctx context.Context, r CommandRunner, volume string) bool {
	if volume == "" {
		return false
	}

	out, err := r.Run(ctx, "diskutil", "apfs", "listSnapshots", volume, "-plist")
	if err != nil {
		return false
	}

	var pl apfsSnapshotsPlist
	if _, err := plist.Unmarshal(out, &pl); err != nil {
		return false
	}

	for _, snap := range pl.Snapshots {
		if tmNameRe.MatchString(snap.SnapshotName) {
			return true
		}
	}

	return false
}

// parseBoolish handles plist values that may be bool or string ("YES"/"NO").
func parseBoolish(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return strings.EqualFold(val, "true") || strings.EqualFold(val, "yes")
	default:
		return false
	}
}
