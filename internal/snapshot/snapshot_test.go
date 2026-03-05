package snapshot

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "valid date",
			input: "2026-03-01-143025",
			want:  time.Date(2026, 3, 1, 14, 30, 25, 0, time.Local),
		},
		{
			name:  "midnight",
			input: "2026-01-15-000000",
			want:  time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local),
		},
		{
			name:  "end of day",
			input: "2026-12-31-235959",
			want:  time.Date(2026, 12, 31, 23, 59, 59, 0, time.Local),
		},
		{
			name:    "too short",
			input:   "2026-03-01",
			wantErr: true,
		},
		{
			name:    "too long",
			input:   "2026-03-01-1430250",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not-a-date-string",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDate(%q) = %v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDate(%q) error = %v", tt.input, err)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ParseDate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Date(2026, 3, 1, 14, 30, 0, 0, time.Local)

	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "seconds",
			t:    now.Add(-30 * time.Second),
			want: "30s",
		},
		{
			name: "zero seconds",
			t:    now,
			want: "0s",
		},
		{
			name: "minutes",
			t:    now.Add(-3 * time.Minute),
			want: "3m",
		},
		{
			name: "hours",
			t:    now.Add(-2 * time.Hour),
			want: "2h",
		},
		{
			name: "days",
			t:    now.Add(-48 * time.Hour),
			want: "2d",
		},
		{
			name: "future",
			t:    now.Add(10 * time.Minute),
			want: "future",
		},
		{
			name: "just under a minute",
			t:    now.Add(-59 * time.Second),
			want: "59s",
		},
		{
			name: "exactly one minute",
			t:    now.Add(-60 * time.Second),
			want: "1m",
		},
		{
			name: "just under an hour",
			t:    now.Add(-59 * time.Minute),
			want: "59m",
		},
		{
			name: "exactly one hour",
			t:    now.Add(-60 * time.Minute),
			want: "1h",
		},
		{
			name: "six days",
			t:    now.Add(-6 * 24 * time.Hour),
			want: "6d",
		},
		{
			name: "one week",
			t:    now.Add(-7 * 24 * time.Hour),
			want: "1w",
		},
		{
			name: "three weeks",
			t:    now.Add(-21 * 24 * time.Hour),
			want: "3w",
		},
		{
			name: "one month",
			t:    now.Add(-30 * 24 * time.Hour),
			want: "1mo",
		},
		{
			name: "three months",
			t:    now.Add(-90 * 24 * time.Hour),
			want: "3mo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRelativeTime(tt.t, now)
			if got != tt.want {
				t.Errorf("FormatRelativeTime(%v, %v) = %q, want %q", tt.t, now, got, tt.want)
			}
		})
	}
}
