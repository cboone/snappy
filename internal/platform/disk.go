package platform

import (
	"context"
	"fmt"
	"strings"
)

// DiskInfo holds formatted disk usage information.
type DiskInfo struct {
	Total     string
	Used      string
	Available string
	Percent   string
}

// String returns a formatted disk info summary.
func (d DiskInfo) String() string {
	return fmt.Sprintf("%s total, %s used, %s available (%s)", d.Total, d.Used, d.Available, d.Percent)
}

// FormatBytes formats a byte count as a human-readable string using binary
// units (1024-based) with conventional labels (e.g. 2153406005248 -> "2.0 TB").
func FormatBytes(b int64) string {
	if b == 0 {
		return "0 B"
	}
	const unit float64 = 1024
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	val := float64(b)
	for i := 1; i < len(units); i++ {
		if val < unit {
			if i == 1 {
				return fmt.Sprintf("%d B", b)
			}
			return fmt.Sprintf("%.1f %s", val, units[i-1])
		}
		val /= unit
	}
	return fmt.Sprintf("%.1f %s", val, units[len(units)-1])
}

// GetDiskInfo runs df -h on the given mount point and returns formatted
// disk usage information.
func GetDiskInfo(ctx context.Context, r CommandRunner, mount string) (DiskInfo, error) {
	out, err := r.Run(ctx, "df", "-h", mount)
	if err != nil {
		return DiskInfo{}, fmt.Errorf("getting disk info: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return DiskInfo{}, fmt.Errorf("unexpected df output")
	}

	// Parse the data line (second line, after header)
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return DiskInfo{}, fmt.Errorf("unexpected df output fields: %d", len(fields))
	}

	return DiskInfo{
		Total:     fields[1],
		Used:      fields[2],
		Available: fields[3],
		Percent:   fields[4],
	}, nil
}
