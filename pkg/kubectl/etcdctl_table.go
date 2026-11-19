package kubectl

import (
	"strings"
)

const etcdCellWidth = 30

func renderEtcdTable(header []string, rows [][]string) string {
	titles := make([]string, len(header))
	widths := make([]int, len(header))
	for i, h := range header {
		titles[i] = strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(h, "_", " ")))
		widths[i] = len(titles[i])
	}
	wrapped := make([][][]string, len(rows))
	for r, row := range rows {
		wrapped[r] = make([][]string, len(row))
		for c, cell := range row {
			wrapped[r][c] = wrapCell(cell, etcdCellWidth)
			for _, line := range wrapped[r][c] {
				widths[c] = max(widths[c], len(line))
			}
		}
	}
	var b strings.Builder
	rule := tableRule(widths)
	b.WriteString(rule)
	b.WriteString(tableLine(titles, widths, centered))
	b.WriteString(rule)
	for _, row := range wrapped {
		writeWrappedRow(&b, row, widths)
	}
	b.WriteString(rule)
	return b.String()
}

func writeWrappedRow(b *strings.Builder, row [][]string, widths []int) {
	height := 1
	for _, lines := range row {
		height = max(height, len(lines))
	}
	for x := 0; x < height; x++ {
		cells := make([]string, len(row))
		for c, lines := range row {
			cells[c] = "  "
			if x < len(lines) {
				cells[c] = lines[x]
			}
		}
		b.WriteString(tableLine(cells, widths, rightAligned))
	}
}

func tableRule(widths []int) string {
	var b strings.Builder
	b.WriteString("+")
	for _, w := range widths {
		b.WriteString(strings.Repeat("-", w+2) + "+")
	}
	return b.String() + "\n"
}

func tableLine(cells []string, widths []int, align func(string, int) string) string {
	var b strings.Builder
	b.WriteString("|")
	for i, cell := range cells {
		b.WriteString(" " + align(cell, widths[i]) + " |")
	}
	return b.String() + "\n"
}

func centered(s string, width int) string {
	gap := width - len(s)
	if gap <= 0 {
		return s
	}
	left := gap / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", gap-left)
}

func rightAligned(s string, width int) string {
	if gap := width - len(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}

func wrapCell(s string, limit int) []string {
	if len(s) <= limit {
		return []string{s}
	}
	lines := []string{}
	current := ""
	for _, word := range strings.Fields(s) {
		switch {
		case current == "":
			current = word
		case len(current)+1+len(word) <= limit:
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
	}
	return append(lines, current)
}
