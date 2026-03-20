package ui

import (
	"fmt"
	"io"
	"text/tabwriter"
)

type TableRow struct {
	Group  string
	App    string
	Status string
	PID    string
}

func PrintTable(w io.Writer, rows []TableRow) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "GROUP\tAPP\tSTATUS\tPID")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Group, r.App, r.Status, r.PID)
	}
	tw.Flush()
}
