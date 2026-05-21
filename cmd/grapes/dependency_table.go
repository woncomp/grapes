package main

import (
	"fmt"
	"strings"

	"github.com/woncomp/grapes/parser"
)

func renderDependencyTable(results []grapeDependencyResult, allowWarnings bool) string {
	rows := [][]string{{"GRAPE", "DEPENDENCY", "STATUS", "LOCATION", "VERSION", "RENDER"}}
	for _, result := range results {
		rows = append(rows, []string{
			grapeDisplayName(result.Grape),
			result.Dependency,
			string(result.Status),
			result.Location,
			result.Version,
			renderDecision(result.Status, allowWarnings),
		})
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var b strings.Builder
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				b.WriteString("  ")
			}
			fmt.Fprintf(&b, "%-*s", widths[i], cell)
		}
		b.WriteByte('\n')
	}

	for _, result := range results {
		if result.Detail == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s\n", grapeDisplayName(result.Grape), result.Detail)
	}
	return b.String()
}

func grapeDisplayName(grape *parser.GrapeFile) string {
	if grape == nil {
		return ""
	}
	if strings.TrimSpace(grape.Label) != "" {
		return grape.Label
	}
	return grape.Name
}

func renderDecision(status dependencyStatus, allowWarnings bool) string {
	switch status {
	case dependencyStatusOK:
		return "yes"
	case dependencyStatusWarning:
		if allowWarnings {
			return "yes"
		}
		return "no"
	default:
		return "no"
	}
}
