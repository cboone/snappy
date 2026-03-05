// Package snapshot provides snapshot types, date parsing, and relative
// time formatting for Time Machine local snapshots.
package snapshot

import (
	"fmt"
	"time"
)

// Snapshot represents a single Time Machine local snapshot.
type Snapshot struct {
	Date         string
	Time         time.Time
	UUID         string
	XID          int
	Purgeable    bool
	LimitsShrink bool
}

// ParseDate parses a Time Machine snapshot date string in YYYY-MM-DD-HHMMSS
// format and returns the corresponding time.Time in the local timezone.
func ParseDate(s string) (time.Time, error) {
	if len(s) != 17 {
		return time.Time{}, fmt.Errorf("invalid snapshot date length: %q", s)
	}
	t, err := time.ParseInLocation("2006-01-02-150405", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid snapshot date %q: %w", s, err)
	}
	return t, nil
}

// FormatRelativeTime returns a human-readable duration since t, such as
// "3m ago", "2h ago", or "1d ago".
func FormatRelativeTime(t, now time.Time) string {
	delta := now.Sub(t)
	if delta < 0 {
		return "future"
	}

	seconds := int(delta.Seconds())
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds ago", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm ago", seconds/60)
	case seconds < 86400:
		return fmt.Sprintf("%dh ago", seconds/3600)
	default:
		return fmt.Sprintf("%dd ago", seconds/86400)
	}
}
