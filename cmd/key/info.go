package main

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// maskValue shortens a value for display: secrets are recognised by their
// prefix, so the rest is hidden from shoulder-surfers and screen shares.
// Values short enough that masking would show nothing useful stay as-is.
func maskValue(v string) string {
	const visible, threshold = 6, 8
	r := []rune(v)
	if len(r) <= threshold {
		return v
	}
	return string(r[:visible]) + "…"
}

func cmdInfo(path string, args []string) {
	reveal := false
	names := make([]string, 0, 1)
	for _, a := range args {
		switch a {
		case "-v", "--reveal":
			reveal = true
		default:
			names = append(names, a)
		}
	}
	if len(names) != 1 {
		usageError("usage: key info [-v] <name>")
	}
	f := load(path)
	e := f.Entry(names[0])
	if e == nil {
		fmt.Fprintf(os.Stderr, "key: %q not found in %s\n", names[0], path)
		os.Exit(1)
	}
	value := e.Value
	if !reveal {
		if masked := maskValue(e.Value); masked != e.Value {
			value = fmt.Sprintf("%s (%d chars)", masked, len(e.Value))
		}
	}
	fmt.Printf("name:      %s\n", e.Name)
	fmt.Printf("value:     %s\n", value)
	fmt.Printf("created:   %s\n", dateCell(e.Created))
	fmt.Printf("modified:  %s\n", dateCell(e.Modified))
	if len(e.Notes) == 0 {
		fmt.Printf("notes:     -\n")
		return
	}
	fmt.Printf("notes:\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, n := range e.Notes {
		fmt.Fprintf(w, "  %s\t%s\n", n.Label, n.Value)
	}
	w.Flush()
}
