package hud

import (
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	GridHome     = "grid:home"
	GridScale    = "grid:scale"
	SelClose     = "sel:close"
	gridPrefix   = "grid"
	GridCells    = 9
	BackCell     = 8
	CentreCell   = 4
	spotLife     = 30.0
	detailsLabel = "DESCRIBE"
)

var (
	backKey     = gridKeyNames[BackCell : BackCell+1]
	homeTip     = "Back to the home grid [Esc, " + backKey + " or right click]"
	homeTipCrew = "Back to the hut's own commands [" + backKey + "]"
)

var GridActions = map[string]string{
	GridScale: ScaleUpButton, "grid:scale-down": ScaleDownButton, "grid:retire": "keep:retire",
	"grid:delete": ActionPrefix + ":" + KeyDelete, "grid:drain": ActionPrefix + ":" + KeyDrain,
	"grid:cordon": ActionPrefix + ":" + KeyCordon, "grid:logs": ActionPrefix + ":" + KeyLogs,
	"grid:berth": BerthButton, "grid:close-berth": CloseBerthButton,
}

type gridSlot struct {
	cell   int
	button Button
}

type gridState struct {
	sel      string
	home     bool
	spot     string
	spotAt   float64
	spotSeen bool
}

func init() {
	prefixHandlers[gridPrefix] = (*HUD).pressGrid
	for alias, id := range GridActions {
		NavAliases[alias] = id
	}
}

func selKey(sel Selection) string {
	if sel.Empty() {
		return ""
	}
	return sel.Kind + "/" + sel.ID + "/" + sel.Structure.ID + "/" + sel.Pod.Name
}

func (h *HUD) syncGrid(f Frame) {
	key := selKey(f.Selection)
	if key == h.grid.sel {
		h.applyPending(f)
		return
	}
	h.grid = gridState{sel: key, spot: h.grid.spot, spotAt: h.grid.spotAt, spotSeen: h.grid.spotSeen}
	h.lw.page = 0
	h.Crew = crewCardState{}
	h.applyPending(f)
	if key != "" {
		h.PageID = PageMain
	}
}

func (h *HUD) objectShown(f Frame) bool {
	return f.cardSelection() && !(h.grid.home && h.grid.sel == selKey(f.Selection)) && !(introAtHome(f) && !f.goalOnSelection())
}

func introAtHome(f Frame) bool {
	return f.Introducing() && placeOf(f.Guide.Intro.Target).page != ""
}

func (h *HUD) GridShown(f Frame) bool { return h.objectShown(f) }

func (h *HUD) objectGrid(f Frame) []Button {
	slots := h.gridSlots(f)
	taken := map[int]bool{}
	out := make([]Button, 0, len(slots)+1)
	for _, s := range slots {
		if s.button.ID == "" || taken[s.cell] || s.cell < 0 || s.cell >= GridCells {
			continue
		}
		taken[s.cell] = true
		b := s.button
		b.Rect = CellRect(s.cell)
		out = append(out, b)
	}
	if !taken[BackCell] {
		b := homeButton()
		b.Rect = CellRect(BackCell)
		out = append(out, b)
	}
	slices.SortStableFunc(out, func(a, b Button) int { return cellOf(a.Rect) - cellOf(b.Rect) })
	return out
}

func cellOf(r ui.Rect) int {
	for i := range GridCells {
		if CellRect(i) == r {
			return i
		}
	}
	return GridCells
}

func homeButton() Button {
	return Button{ID: GridHome, Label: "BACK", Icon: "icon/back", Tip: homeTip}
}

func (h *HUD) gridSlots(f Frame) []gridSlot {
	return h.withLiveOrders(h.cardSlots(f))
}

func (h *HUD) cardSlots(f Frame) []gridSlot {
	switch {
	case HavenSelected(f):
		return portSlots(f)
	case h.crewGrid(f):
		return h.crewSlots(f)
	case f.Selection.Kind == SelectForeman || f.Selection.Kind == SelectCrew:
		return h.foremanSlots(f)
	case hearthSelected(f):
		return h.hutSlots(f)
	case f.Selection.Kind == "structure":
		return h.structureSlots(f)
	}
	return nil
}

func (h *HUD) pressGrid(rest string, f Frame) []Intent {
	if rest != "home" {
		return nil
	}
	return append(h.leaveObject(f), Intent{Kind: IntentSound, Command: "click"})
}

func (h *HUD) leaveObject(f Frame) []Intent {
	h.grid.sel, h.grid.home = selKey(f.Selection), true
	h.PageID = PageMain
	h.Crew = crewCardState{}
	return []Intent{{Kind: IntentSelect}}
}

func (h *HUD) Spotlight(target string) {
	id := target
	if alias, ok := GridActions[target]; ok {
		id = alias
	}
	h.grid.spot, h.grid.spotAt, h.grid.spotSeen = id, h.Now, false
}

func (h *HUD) Spotlit(f Frame) string {
	if h.grid.spot == "" || h.Now-h.grid.spotAt > spotLife {
		return ""
	}
	id := portGoal(h.grid.spot, f)
	if _, ok := HitID(h.CommandButtons(f), id); !ok {
		return ""
	}
	return id
}

func (h *HUD) stepSpot(f Frame) {
	switch lit := h.Spotlit(f) != ""; {
	case lit:
		h.grid.spotSeen = true
	case h.grid.spotSeen || h.Now-h.grid.spotAt > spotLife:
		h.grid.spot, h.grid.spotSeen = "", false
	}
}

func (h *HUD) spotPressed(id string) {
	if id == h.grid.spot || strings.HasPrefix(id, h.grid.spot+":") {
		h.grid.spot, h.grid.spotSeen = "", false
	}
}
