package snapshot

// DiffResult holds the results of comparing two snapshot lists.
type DiffResult struct {
	Added   []Snapshot
	Removed []Snapshot
}

// ComputeDiff compares a previous and current snapshot list and returns
// the snapshots that were added and removed between the two.
func ComputeDiff(prev, curr []Snapshot) DiffResult {
	prevSet := make(map[string]struct{}, len(prev))
	for _, s := range prev {
		prevSet[s.Date] = struct{}{}
	}

	currSet := make(map[string]struct{}, len(curr))
	for _, s := range curr {
		currSet[s.Date] = struct{}{}
	}

	var added []Snapshot
	for _, s := range curr {
		if _, ok := prevSet[s.Date]; !ok {
			added = append(added, s)
		}
	}

	var removed []Snapshot
	for _, s := range prev {
		if _, ok := currSet[s.Date]; !ok {
			removed = append(removed, s)
		}
	}

	return DiffResult{Added: added, Removed: removed}
}
