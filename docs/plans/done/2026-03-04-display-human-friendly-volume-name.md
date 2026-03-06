# 2026-03-04 Display human-friendly volume name in info panel

## Context

The info panel currently shows `Volume: /` (the raw mount point), which is not very informative. macOS volumes have human-friendly names (e.g., "Macintosh HD") available via `diskutil info -plist`. We already call this at startup in `FindAPFSVolume` (via `getDeviceIdentifier`), so the data is readily available.

## Plan

### Files to modify

- `internal/platform/diskutil.go` -- add `VolumeName` to plist struct, add `GetVolumeName` function
- `cmd/root.go` -- resolve volume name at startup, pass to `NewModel`
- `internal/tui/model.go` -- add `volumeName` field, accept it in `NewModel`
- `internal/tui/view.go` -- display volume name instead of raw mount point

### 1. Extend diskutil plist parsing (`internal/platform/diskutil.go`)

Add `VolumeName` to the existing `diskutilInfoPlist` struct:

```go
type diskutilInfoPlist struct {
    DeviceIdentifier string `plist:"DeviceIdentifier"`
    VolumeName       string `plist:"VolumeName"`
}
```

Add an exported `GetVolumeName` function that calls `diskutil info -plist <mount>` and returns the `VolumeName` field:

```go
func GetVolumeName(ctx context.Context, r CommandRunner, mount string) (string, error) {
    // reuse the same plist parsing as getDeviceIdentifier
}
```

### 2. Resolve at startup (`cmd/root.go`)

Call `platform.GetVolumeName` alongside the existing `FindAPFSVolume` call. Pass the result to `NewModel`. Fall back to `cfg.MountPoint` if the call fails.

### 3. Thread through the model (`internal/tui/model.go`)

Add a `volumeName string` parameter to `NewModel` and store it in the model.

### 4. Display in info panel (`internal/tui/view.go`)

Replace `m.cfg.MountPoint` with `m.volumeName` in the Volume line of `renderInfoPanel`.

### Verification

- `make build` compiles cleanly
- `make test` passes
- `make lint` passes
- Run `bin/snappy` and confirm the info panel shows e.g., `Volume: Macintosh HD` instead of `Volume: /`
