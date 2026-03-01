# Add APFS snapshot details to display

## Context

The TUI currently shows only the snapshot date and a relative timestamp per snapshot. The user needs the APFS snapshot UUID to cross-reference snapshots in Disk Utility, and the purgeable flag and other-snapshot count provide useful context for understanding macOS cleanup behavior.

`diskutil apfs listSnapshots <volume> -plist` provides structured per-snapshot data (UUID, Purgeable, XID, Name) that we can correlate with the tmutil date list.

### Key finding from exploration

On macOS, `/` is mounted from a sealed system snapshot (`disk3s1s1`). Time Machine snapshots live on the Data volume (`disk3s5` at `/System/Volumes/Data`). `diskutil apfs listSnapshots disk3s1s1` fails because it's a snapshot mount, not a volume. The correct volume must be discovered at startup.

## Changes to `bin/snappy`

### 1. New global state

```bash
declare -A SNAPSHOT_UUID=()        # date -> full UUID
declare -A SNAPSHOT_PURGEABLE=()   # date -> "Yes" / "No"
declare -i OTHER_SNAPSHOT_COUNT=0  # non-TM snapshots on the volume
APFS_VOLUME=""                     # device identifier (e.g., disk3s5)
```

### 2. New function: `find_apfs_volume()`

Determine the APFS volume to query for snapshot details. Called once at startup.

Strategy:

1. Get the device identifier via `diskutil info -plist <mount_point> | plutil -extract DeviceIdentifier raw -`
2. Try `diskutil apfs listSnapshots <device>` directly (works for external volumes)
3. If that fails and mount point is `/`, try the Data volume at `/System/Volumes/Data`
4. If nothing works, set `APFS_VOLUME=""` and degrade gracefully (no UUID/purgeable columns)

### 3. New function: `apfs_get_snapshot_details()`

Called each refresh cycle after `tm_list_snapshots`. Populates `SNAPSHOT_UUID`, `SNAPSHOT_PURGEABLE`, and `OTHER_SNAPSHOT_COUNT` by parsing `diskutil apfs listSnapshots -plist <APFS_VOLUME>`.

Uses `plutil -extract "Snapshots.<index>.<field>" raw -` in a loop to extract each snapshot's fields. Correlates to tmutil dates by extracting `YYYY-MM-DD-HHMMSS` from the `SnapshotName` field (`com.apple.TimeMachine.YYYY-MM-DD-HHMMSS.local`). Snapshots without TM name prefix increment `OTHER_SNAPSHOT_COUNT`.

Skipped entirely when `APFS_VOLUME` is empty.

### 4. Modified: `draw_header()`

Add an APFS info line between the Time Machine line and the Disk line:

```text
  APFS Volume: disk3s5  |  Other snapshots: 1 (non-Time Machine)
```

Omitted when `APFS_VOLUME` is empty.

### 5. Modified: `draw_snapshot_list()`

Append UUID and purgeable flag to each snapshot line when available:

```text
   1.  2026-02-28-192720   (3m ago)   FE8AFD62-B1C1-409C-A8F9-3FCB96F11325   purgeable
```

When a snapshot has `Purgeable: No`, display "pinned" instead (yellow, to call attention). When APFS details are unavailable, display the current format unchanged.

### 6. Modified: `do_refresh()`

Call `apfs_get_snapshot_details` after `tm_list_snapshots`.

### 7. Modified: `main()`

Call `find_apfs_volume` during initialization (after `check_dependencies`, before the main loop).

## Verification

1. `bash -n bin/snappy` passes
2. `shellcheck bin/snappy` passes clean
3. Run `bin/snappy`, verify UUID and purgeable columns appear for each snapshot
4. Cross-reference a displayed UUID against Disk Utility's APFS snapshot list
5. Press `s` to create a snapshot, verify the new snapshot shows UUID and purgeable on refresh
