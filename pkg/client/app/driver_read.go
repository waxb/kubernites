package app

import (
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	gutterMin     = 12
	columnLines   = 3
	glyphGap      = 2
	wordGap       = 9
	tallRun       = ui.LineHeight * 3 / 2
	tooltipRegion = "tooltip"
)

func PlayText(runs []ui.TextRun) ScreenText {
	order := []string{}
	byRegion := map[string][]ui.TextRun{}
	for _, r := range runs {
		if r.Text == "" {
			continue
		}
		if _, seen := byRegion[r.Region]; !seen {
			order = append(order, r.Region)
		}
		byRegion[r.Region] = append(byRegion[r.Region], r)
	}
	out := ScreenText{}
	for _, name := range order {
		lines := []string{}
		for _, col := range columnsOf(name, byRegion[name]) {
			lines = append(lines, readLines(col)...)
		}
		out.Regions = append(out.Regions, RegionText{Region: name, Lines: lines})
	}
	return out
}

func columnsOf(region string, runs []ui.TextRun) [][]ui.TextRun {
	if !strings.HasPrefix(region, hud.PanelRegion) && region != tooltipRegion {
		return [][]ui.TextRun{runs}
	}
	cut, ok := columnCut(runs)
	if !ok {
		return [][]ui.TextRun{runs}
	}
	var left, right []ui.TextRun
	for _, r := range runs {
		if r.Rect.X < cut {
			left = append(left, r)
		} else {
			right = append(right, r)
		}
	}
	if len(readLines(left)) < columnLines || len(readLines(right)) < columnLines {
		return [][]ui.TextRun{runs}
	}
	return append(columnsOf(region, left), columnsOf(region, right)...)
}

func columnCut(runs []ui.TextRun) (int, bool) {
	spans := make([][2]int, 0, len(runs))
	for _, r := range runs {
		spans = append(spans, [2]int{r.Rect.X, r.Rect.X + r.Rect.W})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i][0] < spans[j][0] })
	best, at := 0, 0
	for i := 1; i < len(spans); i++ {
		if gap := spans[i][0] - spans[i-1][1]; gap > best {
			best, at = gap, spans[i][0]
		}
		spans[i][1] = max(spans[i][1], spans[i-1][1])
	}
	return at, best >= gutterMin
}

type textLine struct {
	top, bottom int
	runs        []ui.TextRun
}

func readLines(runs []ui.TextRun) []string {
	sorted := append([]ui.TextRun(nil), runs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].Rect, sorted[j].Rect
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	lines := []textLine{}
	tall := []ui.TextRun{}
	for _, r := range sorted {
		if r.Rect.H > tallRun {
			tall = append(tall, r)
			continue
		}
		mid := r.Rect.Y + r.Rect.H/2
		if n := len(lines); n > 0 && (Abs(r.Rect.Y-lines[n-1].top) <= lineSlack || mid < lines[n-1].bottom) {
			lines[n-1].runs = append(lines[n-1].runs, r)
			continue
		}
		lines = append(lines, textLine{top: r.Rect.Y, bottom: r.Rect.Y + r.Rect.H, runs: []ui.TextRun{r}})
	}
	for _, r := range tall {
		lines = withTall(lines, r)
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, joinRuns(l.runs))
	}
	return out
}

func withTall(lines []textLine, r ui.TextRun) []textLine {
	for i, l := range lines {
		if l.bottom > r.Rect.Y && l.top < r.Rect.Y+r.Rect.H {
			lines[i].runs = append(append([]ui.TextRun(nil), l.runs...), r)
			return lines
		}
	}
	at := sort.Search(len(lines), func(i int) bool { return lines[i].top > r.Rect.Y })
	return append(lines[:at], append([]textLine{{top: r.Rect.Y, bottom: r.Rect.Y + r.Rect.H, runs: []ui.TextRun{r}}}, lines[at:]...)...)
}

func joinRuns(runs []ui.TextRun) string {
	sort.SliceStable(runs, func(i, j int) bool { return runs[i].Rect.X < runs[j].Rect.X })
	var b strings.Builder
	for i, r := range runs {
		if prev := runs[max(i-1, 0)]; i > 0 && !dropCap(prev) {
			b.WriteString(spaceFor(r.Rect.X - (prev.Rect.X + prev.Rect.W)))
		}
		b.WriteString(r.Spelled)
	}
	return b.String()
}

func dropCap(r ui.TextRun) bool {
	return r.Rect.H > tallRun && len([]rune(r.Text)) == 1
}

func spaceFor(gap int) string {
	switch {
	case gap < glyphGap:
		return ""
	case gap <= wordGap:
		return " "
	}
	return runGap
}
