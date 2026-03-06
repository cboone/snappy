package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestStartupSummaryUsesResolvedVolume(t *testing.T) {
	got := startupSummary("dev", "Macintosh HD", 45*time.Second)

	if !strings.Contains(got, "volume=Macintosh HD") {
		t.Fatalf("startupSummary() missing resolved volume name, got %q", got)
	}
	if !strings.Contains(got, "refresh=45s") {
		t.Fatalf("startupSummary() missing refresh interval, got %q", got)
	}
}
