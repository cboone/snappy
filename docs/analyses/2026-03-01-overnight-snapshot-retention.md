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

## Key finding: macOS thins to one per hour, not a hard TTL

Initial analysis incorrectly concluded that macOS enforces a hard 60-minute TTL
on all snapshots. Closer inspection reveals that macOS thins to approximately
**one snapshot per hour** at the 60-minute boundary, consistent with Time
Machine's documented retention (hourly for 24 hours).

Evidence: at shutdown, hourly survivors were still present at :48/:49 past each
hour going back many hours. Only Snappy's own 6-24h tier removed those hourly
keepers later.

Disk pressure is not a factor: available space held at ~7.0Ti (1% utilization).

## Per-hour lifecycle

Each hour follows a consistent pattern (using hour 21 as an example):

| Step | Count | What happens                                           |
| ---- | ----- | ------------------------------------------------------ |
| 1    | 12    | Snappy creates one snapshot every 5 minutes            |
| 2    | 10    | macOS removes 10 of 12 at the 60-minute mark           |
| 3    | 1     | Snappy's 1-6h tier thins 1 more (racing macOS)         |
| 4    | 1     | One survivor (~:48) persists as the hourly keeper       |
| 5    | 0     | At ~6h, Snappy's 6-24h tier thins the hourly keeper    |

macOS does not touch the hourly keepers. Only Snappy's tiers remove them.

## Thinning tiers work as designed

Both macOS and Snappy are doing complementary thinning:

- **0-1h (Snappy: keep every 5 min):** All 12 snapshots per hour survive.
  Full 5-minute granularity.
- **At 60 min:** macOS thins 10-11 per hour, Snappy thins 1. One hourly
  keeper survives.
- **At 6h (Snappy: keep every 4h):** Snappy removes excess hourly keepers.
  macOS does not intervene.
- **At 24h (Snappy: keep every 1d):** Expected to work similarly, not yet
  observed in this run.

The tiers are not dead letter. They operate in concert with macOS's own
retention policy.

## Steady state

The snapshot count climbed to 13 in the first hour, then slowly drifted up to
19 where it stabilized. Each cycle:

1. Snappy creates a snapshot every 5 minutes (+1)
2. At 60 minutes, macOS or Snappy removes the excess (-1)
3. Net count stays roughly constant (~19 snapshots)

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

Snappy's retention tiers align well with macOS's behavior:

- **Last hour:** full 5-minute granularity (~12 snapshots)
- **Hours 1-6:** one snapshot per hour (macOS and Snappy cooperate)
- **Hours 6-24:** one snapshot per 4 hours (Snappy thins, macOS allows)
- **Days 1-14:** one snapshot per day (expected, not yet observed)

The system works. No code changes are needed for retention.
