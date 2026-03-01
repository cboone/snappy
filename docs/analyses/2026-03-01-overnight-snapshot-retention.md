# Overnight snapshot retention analysis

Analysis of an 11.5-hour overnight run (20:43 Feb 28 to 08:12 Mar 1).

Log file: `logs/2026-03-01-081206-snappy.log`

## Summary

| Metric              | Count |
| ------------------- | ----- |
| Snapshots created   | 137   |
| Thinned (by Snappy) | 14    |
| Removed (by macOS)  | 104   |
| Total disappeared   | 118   |
| Final count         | 19    |
| Errors              | 0     |

## Key finding: macOS enforces a 60-minute TTL

Every one of the 104 system-removed snapshots disappeared at exactly 60 minutes
after creation. Disk pressure is not a factor: available space held steady at
~7.0Ti (1% utilization) throughout the run.

The `deleted` daemon is applying a hard age-based timer to all `tmutil`
snapshots, not a space-pressure heuristic.

One exception: the very first snapshot (`2026-02-28-204827`) survived the entire
run. This is consistent with macOS keeping the oldest snapshot as a baseline
reference.

## Snappy's thinning worked correctly but is mostly preempted

- 10 thinning events at the 1-hour boundary (tier: 1-6h, min gap 1h)
- 4 thinning events at the 6-hour boundary (tier: 6-24h, min gap 4h)

Both tiers operated as designed, but macOS usually gets to the snapshot first.
The retention tiers beyond "0-1h: every 5 min" are effectively dead letter.

## Steady state

The snapshot count climbed to 13 in the first hour, then slowly drifted up to
19 where it stabilized. The pattern each cycle:

1. Snappy creates a snapshot every 5 minutes (+1)
2. 60 minutes later, macOS removes it (-1)
3. Net count stays roughly constant

## LimitingContainerShrink is not relevant

The `LimitingContainerShrink` flag is purely about APFS container resizing. It
identifies which snapshot prevents the container from being shrunk. It has no
effect on purgeability or retention.

## Non-purgeable snapshots require an entitlement

Snapshots created via `tmutil localsnapshot` are always marked `Purgeable: Yes`.
The only way to create non-purgeable snapshots is via the `fs_snapshot_create()`
API, which requires:

- Root privileges
- The `com.apple.developer.vfs.snapshot` entitlement signed into the binary

This entitlement is restricted by Apple and only granted to approved backup
application vendors (Carbon Copy Cloner, Arq, etc.) after code review.

There is no workaround without disabling SIP and AMFI, which is impractical.

## Practical implications

Snappy's current approach (creating every 5 minutes) is the best available
strategy. Even with the 60-minute ceiling, users always have ~12 snapshots
covering the last hour. The thinning tiers beyond 1 hour can remain in the
code for the off chance that macOS behavior varies across versions, but they
will not fire under current conditions.
