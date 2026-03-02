package snapshot

import (
	"testing"
)

func TestComputeDiff(t *testing.T) {
	tests := []struct {
		name        string
		prev        []Snapshot
		curr        []Snapshot
		wantAdded   int
		wantRemoved int
	}{
		{
			name:        "both empty",
			prev:        nil,
			curr:        nil,
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name: "all new",
			prev: nil,
			curr: []Snapshot{
				{Date: "2026-03-01-140000"},
				{Date: "2026-03-01-140100"},
			},
			wantAdded:   2,
			wantRemoved: 0,
		},
		{
			name: "all removed",
			prev: []Snapshot{
				{Date: "2026-03-01-140000"},
				{Date: "2026-03-01-140100"},
			},
			curr:        nil,
			wantAdded:   0,
			wantRemoved: 2,
		},
		{
			name: "no change",
			prev: []Snapshot{
				{Date: "2026-03-01-140000"},
				{Date: "2026-03-01-140100"},
			},
			curr: []Snapshot{
				{Date: "2026-03-01-140000"},
				{Date: "2026-03-01-140100"},
			},
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name: "one added one removed",
			prev: []Snapshot{
				{Date: "2026-03-01-140000"},
				{Date: "2026-03-01-140100"},
			},
			curr: []Snapshot{
				{Date: "2026-03-01-140100"},
				{Date: "2026-03-01-140200"},
			},
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name: "only additions",
			prev: []Snapshot{
				{Date: "2026-03-01-140000"},
			},
			curr: []Snapshot{
				{Date: "2026-03-01-140000"},
				{Date: "2026-03-01-140100"},
				{Date: "2026-03-01-140200"},
			},
			wantAdded:   2,
			wantRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeDiff(tt.prev, tt.curr)
			if len(result.Added) != tt.wantAdded {
				t.Errorf("ComputeDiff() added = %d, want %d", len(result.Added), tt.wantAdded)
			}
			if len(result.Removed) != tt.wantRemoved {
				t.Errorf("ComputeDiff() removed = %d, want %d", len(result.Removed), tt.wantRemoved)
			}
		})
	}
}

func TestComputeDiffDates(t *testing.T) {
	prev := []Snapshot{
		{Date: "2026-03-01-140000"},
		{Date: "2026-03-01-140100"},
	}
	curr := []Snapshot{
		{Date: "2026-03-01-140100"},
		{Date: "2026-03-01-140200"},
	}

	result := ComputeDiff(prev, curr)

	if len(result.Added) != 1 || result.Added[0].Date != "2026-03-01-140200" {
		t.Errorf("expected added 2026-03-01-140200, got %v", result.Added)
	}
	if len(result.Removed) != 1 || result.Removed[0].Date != "2026-03-01-140000" {
		t.Errorf("expected removed 2026-03-01-140000, got %v", result.Removed)
	}
}
