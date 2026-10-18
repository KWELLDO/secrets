package main

import (
	"sort"
	"time"

	"gitee.com/kwelldo/secrets"
)

// staleAfter marks an entry as stale in the list table: a key nobody touched
// for this long is a rotation candidate.
const staleAfter = 90 * 24 * time.Hour

// sortedFor returns a copy of entries ordered by key ("name", "created" or
// "modified"). Dates sort oldest first — unknown (zero) dates count as the
// oldest, since they predate metadata. Ties keep the name order.
func sortedFor(entries []*secrets.Entry, key string) []*secrets.Entry {
	out := make([]*secrets.Entry, len(entries))
	copy(out, entries)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if key == "name" {
		return out
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch key {
		case "modified":
			return older(a.Modified, b.Modified)
		case "created":
			return older(a.Created, b.Created)
		}
		return false
	})
	return out
}

func older(a, b time.Time) bool {
	if a.IsZero() != b.IsZero() {
		return a.IsZero() // unknown dates are the oldest
	}
	return a.Before(b)
}

// modifiedCell renders the MODIFIED column: date plus relative age, and a
// trailing "!" when the entry has not been touched for over staleAfter.
func modifiedCell(e *secrets.Entry, now time.Time) string {
	cell := dateCell(e.Modified)
	if !e.Modified.IsZero() && now.Sub(e.Modified) > staleAfter {
		cell += " !"
	}
	return cell
}
