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

// FormatRelativeTime returns a compact duration since t, such as
// "3m", "2h", "5d", "2w", or "3mo".
func FormatRelativeTime(t, now time.Time) string {
	delta := now.Sub(t)
	if delta < 0 {
		return "future"
	}

	seconds := int(delta.Seconds())
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm", seconds/60)
	case seconds < 86400:
		return fmt.Sprintf("%dh", seconds/3600)
	case seconds < 7*86400:
		return fmt.Sprintf("%dd", seconds/86400)
	case seconds < 30*86400:
		return fmt.Sprintf("%dw", seconds/(7*86400))
	default:
		return fmt.Sprintf("%dmo", seconds/(30*86400))
	}
}
