package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// WriteJSON writes the report as a single indented JSON document.
func WriteJSON(w io.Writer, results []Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(NewReport(results))
}

// WriteText writes human-readable diagnostics followed by a summary line.
func WriteText(w io.Writer, results []Result) error {
	var b strings.Builder
	errs, warns, invalid := 0, 0, 0
	for _, r := range results {
		if r.Valid && len(r.Diagnostics) == 0 {
			fmt.Fprintf(&b, "%s: valid\n", r.Path)
			continue
		}
		if !r.Valid {
			invalid++
		}
		diags := append([]Diagnostic(nil), r.Diagnostics...)
		sort.SliceStable(diags, func(i, j int) bool { return diags[i].Line < diags[j].Line })
		for _, d := range diags {
			loc := d.Path
			if d.Line > 0 {
				loc = fmt.Sprintf("%s:%d", d.Path, d.Line)
			}
			fmt.Fprintf(&b, "%s: %s %s: %s\n", loc, d.Severity, d.Rule, d.Message)
		}
		errs += r.Errors()
		warns += r.Warnings()
	}
	fmt.Fprintf(&b, "\n%s validated, %d valid, %d invalid (%s, %s)\n",
		plural(len(results), "skill"), len(results)-invalid, invalid, plural(errs, "error"), plural(warns, "warning"))
	_, err := io.WriteString(w, b.String())
	return err
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
