package hud

import (
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	PanelCrews    = "crews"
	CrewsKey      = "crews"
	crewsW        = 512
	crewsPad      = 8
	crewsHead     = 28
	crewsRowH     = 16
	crewsRowStep  = 18
	crewsMaxRows  = 8
	crewsBtnW     = 22
	crewsRaiseW   = 64
	crewsCloseKey = CrewsKey + ":close"
)

type CrewsListState struct {
	Open bool `json:"-"`
	sel  string
}

type CrewsRow struct {
	kind     string
	Chain    model.CrewChain `json:"-"`
	trade    RaiseRow
	Standing bool `json:"-"`
}

func (r CrewsRow) Id() string {
	if r.kind != "" {
		return r.kind
	}
	if r.Standing {
		return r.Chain.ID()
	}
	return r.trade.Crew
}

func chainNamed(f Frame, name string) (model.CrewChain, bool) {
	for _, c := range f.Crews {
		if c.Name == name && crewKept(c) {
			return c, true
		}
	}
	return model.CrewChain{}, false
}

func stallCrews(f Frame) []model.CrewChain {
	if !depotSelected(f) || f.Selection.Structure.Object.Kind != serviceKind {
		return nil
	}
	out := []model.CrewChain{}
	for _, c := range f.Crews {
		if crewKept(c) && !c.Bare && c.Depot != "" && c.Depot == f.Selection.Structure.Object.Name {
			out = append(out, c)
		}
	}
	return out
}

func multiStall(f Frame) bool { return len(stallCrews(f)) > 1 }

func CrewsRows(f Frame) []CrewsRow {
	behind := stallCrews(f)
	if len(behind) > 1 {
		out := make([]CrewsRow, 0, len(behind))
		for _, c := range behind {
			out = append(out, CrewsRow{Chain: c, Standing: true})
		}
		return out[:min(len(out), crewsMaxRows)]
	}
	out := []CrewsRow{}
	seen := map[string]bool{}
	for _, t := range RaiseRows(f) {
		seen[t.Crew] = true
		if c, ok := chainNamed(f, t.Crew); ok {
			out = append(out, CrewsRow{Chain: c, trade: t, Standing: true})
			continue
		}
		out = append(out, CrewsRow{trade: t})
	}
	for _, c := range f.Crews {
		if crewKept(c) && !seen[c.Name] {
			out = append(out, CrewsRow{Chain: c, Standing: true})
		}
	}
	out = append(out, kindRows(f)...)
	return out[:min(len(out), crewsMaxRows)]
}

func standingCrews(f Frame) bool {
	for _, c := range f.Crews {
		if crewKept(c) {
			return true
		}
	}
	return false
}

func crewsRect(f Frame, rows int) ui.Rect {
	height := crewsHead + rows*crewsRowStep + crewsPad
	return f.placed(PanelCrews, ui.Rect{X: CardBox.X, Y: ConsoleY - crewFloat - height, W: crewsW, H: height})
}

func (h *HUD) crewsShown(f Frame) bool {
	if h.OverlayID != OverlayNone || h.introCovers(f) {
		return false
	}
	if !(h.CrewsList.Open || multiStall(f)) || len(CrewsRows(f)) == 0 {
		return false
	}
	return len(h.CrewRects(f)) == 0
}

func crewsWindow(h *HUD, f Frame) (ui.Rect, bool) {
	if !h.crewsShown(f) {
		return ui.Rect{}, false
	}
	return crewsRect(f, len(CrewsRows(f))), true
}

func crewsRowRect(box ui.Rect, i int) ui.Rect {
	return ui.Rect{X: box.X + crewsPad, Y: box.Y + crewsHead + i*crewsRowStep, W: box.W - 2*crewsPad, H: crewsRowH}
}

func (h *HUD) crewsButtons(f Frame) []Button {
	if !h.crewsShown(f) {
		return nil
	}
	rows := CrewsRows(f)
	box := crewsRect(f, len(rows))
	out := []Button{closeButtonAs(box, crewsCloseKey, "")}
	for i, r := range rows {
		rr := crewsRowRect(box, i)
		if r.kind != "" {
			b := kindRowButton(rr, r)
			b.Selected = b.ID == f.Guide.Button || r.kind == archiveCardID && h.verbAwaits(f)
			out = append(out, b)
			continue
		}
		if !r.Standing {
			out = append(out, raiseRowButton(rr, r))
			continue
		}
		out = append(out, Button{ID: CrewsKey + ":open:" + r.Id(), Rect: ui.Rect{X: rr.X, Y: rr.Y, W: rr.W - 2*crewsBtnW - 4, H: rr.H}, Tip: openTip(r.Chain)})
		out = append(out, crewsScaleButtons(rr, r, f)...)
	}
	return out
}

func openTip(c model.CrewChain) string {
	if c.Bare {
		return ui.Bridge("Open the "+c.Name+" shift's board: its count, its resources and its land.", "Open the ReplicaSet "+c.Name+": replicas, requests and Nodes.")
	}
	return ui.Bridge("Open the "+c.Name+" crew's board: its slider, its resources and its land.", "Open the Deployment "+c.Name+": replicas, requests and Nodes.")
}

func kindRowButton(rr ui.Rect, r CrewsRow) Button {
	if r.kind == archiveRaiseID {
		return Button{ID: CrewsKey + ":raise-archive", Rect: ui.Rect{X: rr.X + rr.W - crewsRaiseW, Y: rr.Y, W: crewsRaiseW, H: rr.H}, Label: ui.Bridge("RAISE", "CREATE"),
			Tip: ui.Bridge("Raise the Muniment Room: one archivist writes every seal in its books, with the books loose in its hut (an emptyDir), and a headless Service names it.",
				"Create the headless Service archive (clusterIP: None) and the Deployment archive: one archivist:1.0 Pod with its books in an emptyDir volume.")}
	}
	if r.kind == DsRaiseID {
		return Button{ID: CrewsKey + ":raise-ds", Rect: ui.Rect{X: rr.X + rr.W - crewsRaiseW, Y: rr.Y, W: crewsRaiseW, H: rr.H}, Label: ui.Bridge("RAISE", "CREATE"),
			Tip: ui.Bridge("Raise the post-wardens as one crew: a warden on every Keep, now and on every Keep you buy later.", "Create a DaemonSet warden: one Pod on every node, now and on every new node.")}
	}
	return Button{ID: CrewsKey + ":open:" + r.kind, Rect: rr, Tip: ui.Bridge("Open its card: its counts, its Keeps and its describe.", "Open its card: status, Pods and describe.")}
}

func raiseRowButton(rr ui.Rect, r CrewsRow) Button {
	b := Button{ID: CrewsKey + ":raise:" + r.trade.Crew, Rect: ui.Rect{X: rr.X + rr.W - crewsRaiseW, Y: rr.Y, W: crewsRaiseW, H: rr.H},
		Label: ui.Bridge("RAISE", "CREATE"), Disabled: len(r.trade.Images) == 0}
	b.Tip = ui.Bridge("Raise a crew for "+r.trade.Name+": pick its class and its count.", "Create a Deployment for "+r.trade.Name+": pick its image and replicas.")
	if b.Disabled {
		b.Tip = "The Academy trains no class of " + r.trade.Crew + " for you yet: endow one first (ACADEMY)."
	}
	return b
}

func crewsScaleButtons(rr ui.Rect, r CrewsRow, f Frame) []Button {
	pair := ScaleButtons(r.Chain, f)
	less, more := pair[0], pair[1]
	less.ID, more.ID = CrewsKey+":scale:"+r.Id()+":down", CrewsKey+":scale:"+r.Id()+":up"
	less.Label, more.Label = "-", "+"
	less.Selected, more.Selected = false, false
	less.Rect = ui.Rect{X: rr.X + rr.W - 2*crewsBtnW - 2, Y: rr.Y, W: crewsBtnW, H: rr.H}
	more.Rect = ui.Rect{X: rr.X + rr.W - crewsBtnW, Y: rr.Y, W: crewsBtnW, H: rr.H}
	return []Button{less, more}
}

func (h *HUD) PressCrews(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	verb, arg, _ := strings.Cut(rest, ":")
	switch verb {
	case "close":
		h.CrewsList = CrewsListState{}
		return []Intent{click}
	case "open":
		return h.openFromList(arg, f, click)
	case "scale":
		return crewsScale(arg, f, click)
	case "raise-ds":
		h.CrewsList = CrewsListState{}
		return []Intent{ActFrom(view.Action{Action: view.ActRaiseDaemonSet}, FromButton), click}
	case "raise-archive":
		h.CrewsList = CrewsListState{}
		return []Intent{ActFrom(view.Action{Action: view.ActRaiseArchive}, FromButton), click}
	case "raise":
		h.CrewsList = CrewsListState{}
		h.Ops.Raise = h.Ops.Raise.with(arg, nil, nil)
		h.OverlayID = OverlayRaise
		return []Intent{click}
	}
	return nil
}

func (h *HUD) openFromList(id string, f Frame, click Intent) []Intent {
	if _, ok := KindCard(f, id); ok {
		h.CrewsList, h.KindCard = CrewsListState{}, id
		return []Intent{click}
	}
	c, ok := chainByID(f, id)
	if !ok {
		return nil
	}
	h.CrewsList = CrewsListState{}
	sel := CrewSelection(c)
	return []Intent{{Kind: IntentChoose, Target: sel.Kind, Command: sel.ID}, click}
}

func crewsScale(arg string, f Frame, click Intent) []Intent {
	i := strings.LastIndex(arg, ":")
	if i < 0 {
		return nil
	}
	c, ok := chainByID(f, arg[:i])
	if !ok {
		return nil
	}
	a, ok := crewAction(c, "scale", arg[i+1:])
	if !ok {
		return nil
	}
	return []Intent{ActFrom(a, FromCrewCard), click}
}

func (h *HUD) toggleCrews(f Frame) []Intent {
	click := []Intent{{Kind: IntentSound, Command: "click"}}
	if h.CrewsList.Open {
		h.CrewsList = CrewsListState{}
		return click
	}
	h.CrewsList = CrewsListState{Open: true, sel: selKey(f.Selection)}
	return click
}

func (h *HUD) forgetCrewsList(f Frame) {
	if h.CrewsList.Open && selKey(f.Selection) != h.CrewsList.sel {
		h.CrewsList = CrewsListState{}
	}
}

func creatableRows(rows []RaiseRow) []RaiseRow {
	return slices.DeleteFunc(slices.Clone(rows), func(r RaiseRow) bool { return r.Standing })
}
