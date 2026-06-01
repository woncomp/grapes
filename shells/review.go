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
	ops := diffOps(currentLines, proposedLines)

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", path)
	fmt.Fprintf(&b, "+++ %s\n", path)

	const contextLines = 8
	hunks := buildHunks(ops, contextLines)
	for _, h := range hunks {
		oldStart, oldEnd := h.oldRange()
		newStart, newEnd := h.newRange()
		oldCount := max(oldEnd-oldStart+1, 1)
		newCount := max(newEnd-newStart+1, 1)
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart+1, oldCount, newStart+1, newCount)
		for _, op := range h.ops {
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
	}
	return b.String()
}

type hunk struct {
	ops              []diffOp
	oldStart, oldEnd int
	newStart, newEnd int
}

func (h hunk) oldRange() (int, int) { return h.oldStart, h.oldEnd }
func (h hunk) newRange() (int, int) { return h.newStart, h.newEnd }

func buildHunks(ops []diffOp, contextLines int) []hunk {
	type changeRange struct{ start, end int }
	var changes []changeRange
	for i, op := range ops {
		if op.kind != diffEqual {
			changes = append(changes, changeRange{i, i})
		}
	}
	if len(changes) == 0 {
		return nil
	}

	var ranges []changeRange
	for _, c := range changes {
		start := max(c.start-contextLines, 0)
		end := min(c.end+contextLines, len(ops)-1)
		if len(ranges) > 0 && start <= ranges[len(ranges)-1].end+1 {
			ranges[len(ranges)-1].end = end
		} else {
			ranges = append(ranges, changeRange{start, end})
		}
	}

	type indexedOp struct {
		op           diffOp
		oldIdx, newIdx int
	}
	var allIndexed []indexedOp
	oldIdx, newIdx := 0, 0
	for _, op := range ops {
		io := indexedOp{op: op}
		switch op.kind {
		case diffEqual:
			io.oldIdx = oldIdx
			io.newIdx = newIdx
			oldIdx++
			newIdx++
		case diffDelete:
			io.oldIdx = oldIdx
			oldIdx++
		case diffAdd:
			io.newIdx = newIdx
			newIdx++
		}
		allIndexed = append(allIndexed, io)
	}

	var hunks []hunk
	for _, r := range ranges {
		h := hunk{oldStart: -1, newStart: -1}
		for i := r.start; i <= r.end; i++ {
			io := allIndexed[i]
			h.ops = append(h.ops, io.op)
			switch io.op.kind {
			case diffEqual:
				if h.oldStart < 0 {
					h.oldStart = io.oldIdx
				}
				h.oldEnd = io.oldIdx
				if h.newStart < 0 {
					h.newStart = io.newIdx
				}
				h.newEnd = io.newIdx
			case diffDelete:
				if h.oldStart < 0 {
					h.oldStart = io.oldIdx
				}
				h.oldEnd = io.oldIdx
			case diffAdd:
				if h.newStart < 0 {
					h.newStart = io.newIdx
				}
				h.newEnd = io.newIdx
			}
		}
		hunks = append(hunks, h)
	}
	return hunks
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
