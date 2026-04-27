package shells

import (
	"fmt"
	"strings"
)

type Review struct {
	RCFile   string
	Current  string
	Proposed string
	Changed  bool
}

func Preview(rcFile string, installLines []string) (Review, error) {
	current, err := readRCFile(rcFile)
	if err != nil {
		return Review{}, err
	}

	proposed := installedContent(current, installLines)
	return Review{
		RCFile:   rcFile,
		Current:  current,
		Proposed: proposed,
		Changed:  current != proposed,
	}, nil
}

func (r Review) Diff() string {
	if !r.Changed {
		return ""
	}
	return unifiedDiff(r.RCFile, r.Current, r.Proposed)
}

type diffKind int

const (
	diffEqual diffKind = iota
	diffDelete
	diffAdd
)

type diffOp struct {
	kind diffKind
	line string
}

func unifiedDiff(path, current, proposed string) string {
	currentLines := contentLines(current)
	proposedLines := contentLines(proposed)

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", path)
	fmt.Fprintf(&b, "+++ %s\n", path)
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", diffStart(currentLines), len(currentLines), diffStart(proposedLines), len(proposedLines))
	for _, op := range diffOps(currentLines, proposedLines) {
		prefix := ' '
		switch op.kind {
		case diffDelete:
			prefix = '-'
		case diffAdd:
			prefix = '+'
		}
		b.WriteRune(prefix)
		b.WriteString(op.line)
		b.WriteByte('\n')
	}
	return b.String()
}

func contentLines(content string) []string {
	if content == "" {
		return nil
	}
	trimmed := strings.TrimSuffix(content, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func diffStart(lines []string) int {
	if len(lines) == 0 {
		return 0
	}
	return 1
}

func diffOps(current, proposed []string) []diffOp {
	lcs := make([][]int, len(current)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(proposed)+1)
	}

	for i := len(current) - 1; i >= 0; i-- {
		for j := len(proposed) - 1; j >= 0; j-- {
			if current[i] == proposed[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
				continue
			}
			if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var ops []diffOp
	for i, j := 0, 0; i < len(current) || j < len(proposed); {
		switch {
		case i < len(current) && j < len(proposed) && current[i] == proposed[j]:
			ops = append(ops, diffOp{kind: diffEqual, line: current[i]})
			i++
			j++
		case j < len(proposed) && (i == len(current) || lcs[i][j+1] > lcs[i+1][j]):
			ops = append(ops, diffOp{kind: diffAdd, line: proposed[j]})
			j++
		case i < len(current):
			ops = append(ops, diffOp{kind: diffDelete, line: current[i]})
			i++
		}
	}

	return ops
}
