package main

import (
	"strings"
	"testing"
	"time"

	"gitee.com/kwelldo/secrets"
)

var fixedNow = time.Date(2026, 10, 18, 12, 0, 0, 0, time.Local)

func entry(name string, modified time.Time) *secrets.Entry {
	return &secrets.Entry{Name: name, Modified: modified}
}

func names(entries []*secrets.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Name
	}
	return out
}

func TestSortedForModifiedOldestFirst(t *testing.T) {
	old := fixedNow.Add(-100 * 24 * time.Hour)
	newer := fixedNow.Add(-time.Hour)
	in := []*secrets.Entry{entry("new", newer), entry("old", old), entry("unknown", time.Time{})}
	got := names(sortedFor(in, "modified"))
	want := []string{"unknown", "old", "new"} // zero dates are the oldest
	if len(got) != len(want) {
		t.Fatalf("sortedFor = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sortedFor = %v, want %v", got, want)
			break
		}
	}
}

func TestSortedForStableOnEqualDates(t *testing.T) {
	in := []*secrets.Entry{entry("b", fixedNow), entry("a", fixedNow)}
	if got := names(sortedFor(in, "modified")); got[0] != "a" || got[1] != "b" {
		t.Errorf("equal dates must keep name order, got %v", got)
	}
}

func TestSortedForNameDefault(t *testing.T) {
	in := []*secrets.Entry{entry("c", fixedNow), entry("a", fixedNow), entry("b", fixedNow)}
	if got := names(sortedFor(in, "name")); got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("name sort = %v", got)
	}
}

func TestModifiedCellMarksStale(t *testing.T) {
	stale := entry("stale", fixedNow.Add(-91*24*time.Hour))
	if cell := modifiedCell(stale, fixedNow); !strings.HasSuffix(cell, "!") {
		t.Errorf("91d old must be marked stale, got %q", cell)
	}

	fresh := entry("fresh", fixedNow.Add(-time.Hour))
	if cell := modifiedCell(fresh, fixedNow); strings.HasSuffix(cell, "!") {
		t.Errorf("1h old must not be marked stale, got %q", cell)
	}

	unknown := entry("unknown", time.Time{})
	if cell := modifiedCell(unknown, fixedNow); strings.HasSuffix(cell, "!") {
		t.Errorf("unknown date must not be marked stale, got %q", cell)
	}
}
