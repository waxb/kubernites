package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayRoad = "road"

	RoadDone  = "done"
	RoadHere  = "here"
	RoadAhead = "ahead"

	roadInset      = 10
	roadGap        = 8
	roadLeftW      = 612
	roadColTop     = 40
	roadColH       = 120
	roadPathY      = 54
	medalR         = 11
	roadNameY      = 72
	roadTagY       = 83
	roadGoalLabelY = 98
	roadGoalY      = 109
	roadGoalLines  = 4
	roadPaneY      = 168
	roadPaneRows   = 14
	roadPaneCols   = 2
	roadRightGap   = 14
	roadLedgerY    = 40
	roadLedgerRows = 22
	roadLoomW      = 150
)

var RoadBox = ui.Rect{X: 6, Y: TopH + 12, W: Width - 12, H: Overlay.H}

var roadStatuses = map[string]string{
	"done": RoadDone, "past": RoadDone, "complete": RoadDone, "completed": RoadDone,
	"current": RoadHere, "now": RoadHere, "active": RoadHere,
}

var fallbackKeeps = []string{"Grimoire pages", "Academy classes", "words and Chronicles", "the Tapestry of decrees", "Golden Threads and the Loom"}

var fallbackLoses = []string{"gold, Keeps and islands", "berths and renown", "hand-cast huts"}

type roadLines struct {
	name string
	tag  string
	goal []string
}

func roadKnown(f Frame) bool {
	return f.Revealed("ageup") && len(f.State.Road.Ages) > 0
}

func RoadStatus(a view.AgeStep, current string) string {
	if current != "" && a.ID == current {
		return RoadHere
	}
	if s, ok := roadStatuses[strings.ToLower(a.Status)]; ok {
		return s
	}
	return RoadAhead
}

func roadTag(status string) (string, color.RGBA) {
	switch status {
	case RoadDone:
		return "[x] WALKED", ui.Palette.Green
	case RoadHere:
		return "> YOU ARE HERE", ui.Palette.GoldBright
	}
	return "AHEAD", ui.Palette.TextDim
}

func RoadColumns(n int) []ui.Rect {
	if n <= 0 {
		return nil
	}
	w := (roadLeftW - roadGap*(n-1)) / n
	out := make([]ui.Rect, n)
	for i := range out {
		out[i] = ui.Rect{X: RoadBox.X + roadInset + i*(w+roadGap), Y: RoadBox.Y + roadColTop, W: w, H: roadColH}
	}
	return out
}

func roadRight() ui.Rect {
	x := RoadBox.X + roadInset + roadLeftW + roadRightGap
	return ui.Rect{X: x, Y: RoadBox.Y + roadLedgerY, W: RoadBox.X + RoadBox.W - roadInset - x, H: roadLedgerRows * ui.LineHeight}
}

func roadPane() ui.Rect {
	return ui.Rect{X: RoadBox.X + roadInset, Y: RoadBox.Y + roadPaneY, W: roadLeftW, H: (roadPaneRows + 1) * ui.LineHeight}
}

func bulletRows(item string, width int) []string {
	var font ui.Font
	rows := font.Wrap(item, width-2*ui.Advance, 1)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = "  " + r
		if i == 0 {
			out[i] = "- " + r
		}
	}
	return out
}

func RoadUnlockLines(items []string, width, maxLines int) []string {
	all := []string{}
	for _, item := range items {
		all = append(all, bulletRows(item, width)...)
	}
	if len(all) <= maxLines {
		return all
	}
	out := []string{}
	for i, item := range items {
		rows := bulletRows(item, width)
		if len(out)+len(rows) > maxLines-1 {
			return append(out, fmt.Sprintf("+%d more", len(items)-i))
		}
		out = append(out, rows...)
	}
	return out
}

func RoadPaneColumns(items []string, width, rows, cols int) [][]string {
	out := make([][]string, cols)
	col := 0
	for i, item := range items {
		lines := bulletRows(item, width)
		if len(out[col])+len(lines) > rows && col < cols-1 {
			col++
		}
		room := rows - len(out[col])
		if col == cols-1 && i < len(items)-1 {
			room--
		}
		if len(lines) > room {
			out[col] = append(out[col], fmt.Sprintf("+%d more", len(items)-i))
			return out
		}
		out[col] = append(out[col], lines...)
	}
	return out
}

func roadText(a view.AgeStep, status string, width int) roadLines {
	var font ui.Font
	tag, _ := roadTag(status)
	return roadLines{
		name: ui.Clip(strings.ToUpper(a.Name), ui.MaxChars(width, 1)),
		tag:  ui.Clip(tag, ui.MaxChars(width, 1)),
		goal: font.WrapLines(a.Goal, width, 1, roadGoalLines),
	}
}

func roadTip(a view.AgeStep) string {
	return a.Name + ": " + a.Goal + ". Click to list what it unlocks."
}

func ReforgeLedger(f Frame) ([]string, []string) {
	keeps, loses := f.State.Road.ReforgeKeeps, f.State.Road.ReforgeLoses
	if len(keeps) == 0 {
		keeps = fallbackKeeps
	}
	if len(loses) == 0 {
		loses = fallbackLoses
	}
	return keeps, loses
}

func LedgerLines(keeps, loses []string, width, rows int) ([]string, []string) {
	half := rows / 2
	k := RoadUnlockLines(keeps, width, max(rows-min(len(loses), half)-1, half))
	return k, RoadUnlockLines(loses, width, rows-len(k)-1)
}

func SealsText(sg view.SiegeView) string {
	return fmt.Sprintf("%d OF %d SEALS", sg.Seals, sealsGoal(sg))
}

func sealsGoal(sg view.SiegeView) int {
	if sg.SealsGoal <= 0 {
		return 8
	}
	return sg.SealsGoal
}

func roadProgress(f Frame) (string, bool) {
	reqs := f.State.AgeUp.Requirements
	if f.State.AgeUp.NextAge == "" || len(reqs) == 0 {
		return "", false
	}
	met := 0
	for _, r := range reqs {
		if r.Met {
			met++
		}
	}
	return fmt.Sprintf("%d OF %d MET", met, len(reqs)), met == len(reqs)
}

func (h *HUD) selectedAge(f Frame) (view.AgeStep, bool) {
	road := f.State.Road
	for _, id := range []string{h.roadSel, road.Current} {
		for _, a := range road.Ages {
			if a.ID == id && id != "" {
				return a, true
			}
		}
	}
	if len(road.Ages) > 0 {
		return road.Ages[0], true
	}
	return view.AgeStep{}, false
}

func roadLoomButton() Button {
	return Button{ID: "road:loom", Rect: ui.Rect{X: RoadBox.X + RoadBox.W - 8 - roadLoomW, Y: RoadBox.Y + RoadBox.H - 26, W: roadLoomW, H: 18},
		Label: ui.Bridge("THE AGE AND THE LOOM", "PROMOTION AND PRESTIGE"), Tip: "Back to this Age's requirements and the Loom [Esc]"}
}

func roadButtons(f Frame) []Button {
	ages := f.State.Road.Ages
	out := []Button{}
	for i, c := range RoadColumns(len(ages)) {
		out = append(out, Button{ID: "road:age:" + ages[i].ID, Rect: ui.Rect{X: c.X, Y: c.Y, W: c.W, H: roadTagY + ui.LineHeight - roadColTop}, Tip: roadTip(ages[i])})
	}
	out = append(out, Button{ID: "road:seals", Rect: sealsRect(f), Tip: SealsTip})
	return append(out, roadLoomButton(), closeButton(RoadBox))
}

func (h *HUD) roadOverlayButtons(f Frame) []Button { return roadButtons(f) }

func (h *HUD) pressRoad(rest string, _ Frame) []Intent {
	switch {
	case rest == "loom":
		h.OverlayID, h.roadBack = OverlayAge, ""
	case strings.HasPrefix(rest, "age:"):
		h.roadSel = strings.TrimPrefix(rest, "age:")
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) pressAge(rest string, _ Frame) []Intent {
	if rest == "road" {
		h.OverlayID, h.roadBack, h.roadSel = OverlayRoad, OverlayAge, ""
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) toggleRoad() {
	if h.OverlayID == OverlayRoad {
		h.OverlayID, h.roadBack = OverlayNone, ""
		return
	}
	h.roadBack, h.roadSel = "", ""
	if h.OverlayID == OverlayAge {
		h.roadBack = OverlayAge
	}
	h.OverlayID = OverlayRoad
}

func roadAgeButton(f Frame) (Button, bool) {
	if len(f.State.Road.Ages) == 0 {
		return Button{}, false
	}
	b := AgeBottom()
	return Button{ID: "age:road", Rect: ui.Rect{X: b.X + b.W - 204 - 4 - 104, Y: b.Y + 2, W: 104, H: 18}, Label: "ROAD OF AGES",
		Tip: "Every Age, its goal and what it unlocks, and what a Reforge keeps [R]"}, true
}
