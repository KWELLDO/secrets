package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"gitee.com/kwelldo/secrets"
)

// match is one search hit: the entry plus the field that matched. An empty
// field means the name matched.
type match struct {
	entry *secrets.Entry
	field string
}

// searchEntries finds entries whose name or whose note labels/values contain
// q (case-insensitive). Secret values are deliberately never searched.
func searchEntries(entries []*secrets.Entry, q string) []match {
	needle := strings.ToLower(q)
	var out []match
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name), needle) {
			out = append(out, match{entry: e})
		}
		for _, n := range e.Notes {
			if strings.Contains(strings.ToLower(n.Label), needle) ||
				strings.Contains(strings.ToLower(n.Value), needle) {
				out = append(out, match{entry: e, field: n.Label + "=" + n.Value})
			}
		}
	}
	return out
}

// cmdSearch prints entries whose name or notes match; like grep it prints
// nothing and exits 1 when there is no match.
func cmdSearch(path string, args []string) {
	if len(args) != 1 {
		usageError("usage: key search <text>")
	}
	f := load(path)
	matches := searchEntries(f.Entries(), args[0])
	if len(matches) == 0 {
		os.Exit(1)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, m := range matches {
		fmt.Fprintf(w, "%s\t%s\n", m.entry.Name, m.field)
	}
	w.Flush()
}
