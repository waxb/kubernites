package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type gridCase struct {
	name string
	f    Frame
	prep func(h *HUD, f Frame)
}

func fullCrewFrame(sel Selection) Frame {
	f := crewFrame(sel, "soulvault")
	f.State.Unlocks[HornUnlock] = true
	f.State.Grimoire = []view.GrimoirePage{runes("resources.requests.cpu", "resources.requests.memory", "resources.limits.cpu", "resources.limits.memory")}
	f.State.Soulstones = append(f.State.Soulstones, view.SoulstoneEntry{Image: "porter:2.1", Owned: true})
	return f
}

func pressing(ids ...string) func(h *HUD, f Frame) {
	return func(h *HUD, f Frame) {
		for _, id := range ids {
			h.Press(id, f)
		}
	}
}

func gridCases(t *testing.T) []gridCase {
	out := []gridCase{}
	for _, c := range cardCases(t) {
		out = append(out, gridCase{name: c.name, f: c.f})
	}
	hut := Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")}
	offers := offerFrame(t, raiseClerks)
	offers.Selection = Selection{Kind: "structure", ID: "d1", Structure: view.Structure{ID: "d1", Kind: "depot-ledger", State: view.StateReady,
		Object: view.ObjectRef{Kind: "Service", Name: "ledgers"}}}
	chancery := offerFrame(t, raiseClerks)
	chancery = selectStructure(chancery, view.Structure{ID: "c", Kind: view.StructChancery, State: view.StateReady})
	stalls := chooserFrame(t)
	stalls.State.Crews = append(stalls.State.Crews, view.CrewView{Name: "clerks", Replicas: 2})
	stalls.State.Freight.Kinds = append(stalls.State.Freight.Kinds, view.KindView{Kind: "dispatch", Name: "dispatches", Open: true, Depot: "dispatches"})
	stalls.State.Crews = append(stalls.State.Crews, view.CrewView{Name: "couriers", Replicas: 2})
	bare := fullCrewFrame(Selection{Kind: SelectForeman, ID: "default/porters-7d9f"})
	bareChain := crewChain()
	bareChain.Bare = true
	bare.Crews = []model.CrewChain{bareChain}
	return append(out,
		gridCase{name: "crew", f: fullCrewFrame(crewSel())},
		gridCase{name: "crew rollout", f: fullCrewFrame(crewSel()), prep: pressing(CrewModeOpen + CrewModeRoll)},
		gridCase{name: "crew upgrade", f: fullCrewFrame(crewSel()), prep: pressing(CrewModeOpen + CrewModeImage)},
		gridCase{name: "crew resources", f: fullCrewFrame(crewSel()), prep: pressing(CrewModeOpen + CrewModeRes)},
		gridCase{name: "crew horn", f: fullCrewFrame(crewSel()), prep: pressing(CrewModeOpen + CrewModeHorn)},
		gridCase{name: "hut", f: fullCrewFrame(hut)},
		gridCase{name: "hut crew", f: fullCrewFrame(hut), prep: pressing(ChainButton)},
		gridCase{name: "foreman", f: fullCrewFrame(Selection{Kind: SelectForeman, ID: "default/porters-7d9f"})},
		gridCase{name: "bare foreman", f: bare},
		gridCase{name: "harbor", f: harborFrame(t)},
		gridCase{name: "depot offers", f: offers},
		gridCase{name: "chancery offer", f: chancery},
		gridCase{name: "build stalls", f: stalls, prep: func(h *HUD, _ Frame) { h.PageID = PageBuild }},
	)
}

func labelLost(label string, width int) string {
	named := ui.Name(label)
	rows := CellLabel(label, width)
	joined := rows[0]
	if len(rows) == 2 {
		joined = rows[1] + " " + rows[0]
	}
	for _, row := range rows {
		if len([]rune(row)) > width {
			return row
		}
	}
	if len(rows) > 2 || strings.TrimSpace(joined) != strings.TrimSpace(named) {
		return strings.Join(rows, "|")
	}
	return ""
}

func TestEveryGridLabelFitsItsCellInBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	width := ui.MaxChars(CellW-4, 1)
	for _, mode := range namesModes() {
		mode.apply()
		for _, c := range gridCases(t) {
			h := New(nil, (&fakeRunner{}).backend())
			if c.prep != nil {
				c.prep(h, c.f)
			}
			for _, b := range h.CommandButtons(c.f) {
				if lost := labelLost(b.Label, width); lost != "" {
					t.Errorf("%s [%s]: %s reads %q in its cell", c.name, mode.name, b.ID, lost)
				}
				if b.Tip == "" {
					t.Errorf("%s [%s]: %s has no tip", c.name, mode.name, b.ID)
				}
			}
		}
	}
}

func TestEveryGridCommandNamesItsKubectl(t *testing.T) {
	for _, c := range gridCases(t) {
		h := New(nil, (&fakeRunner{}).backend())
		if c.prep != nil {
			c.prep(h, c.f)
		}
		for _, b := range h.CommandButtons(c.f) {
			acts := strings.HasPrefix(b.ID, ActionPrefix+":") || strings.HasPrefix(b.ID, linePrefix+":") || strings.HasPrefix(b.ID, stallPrefix+":")
			orders := strings.HasPrefix(b.ID, CrewPrefix+":scale:") || strings.HasPrefix(b.ID, CrewPrefix+":roll:") || strings.HasPrefix(b.ID, CrewPrefix+":image:")
			if (acts || orders) && !b.Menu && !b.Disabled && !strings.Contains(b.Tip, "kubectl") && !strings.Contains(b.Tip, "etcdctl") {
				t.Errorf("%s: %s tip %q names no command", c.name, b.ID, b.Tip)
			}
		}
	}
}

func TestTheMenuMarkNeverSitsOnTheLabelOrTheCorners(t *testing.T) {
	defer ui.SetNamer(nil)
	cell := CellRect(0)
	labelBand := ui.Rect{X: cell.X, Y: cell.Y + cell.H - 11 - 9, W: cell.W, H: 20}
	for _, m := range menuMarkRects(cell) {
		if m.X < cell.X || m.Y < cell.Y || m.X+m.W > cell.X+cell.W || m.Y+m.H > cell.Y+cell.H {
			t.Errorf("mark %+v leaves its cell %+v", m, cell)
		}
		if m.X < labelBand.X+labelBand.W && labelBand.X < m.X+m.W && m.Y < labelBand.Y+labelBand.H && labelBand.Y < m.Y+m.H {
			t.Errorf("mark %+v sits on the label band %+v, and reads as a clipped label", m, labelBand)
		}
		if m.X+m.W > cell.X+cell.W/2-8 {
			t.Errorf("mark %+v reaches the icon or the top-right badges", m)
		}
	}
	for _, mode := range namesModes() {
		mode.apply()
		for _, c := range gridCases(t) {
			h := New(nil, (&fakeRunner{}).backend())
			if c.prep != nil {
				c.prep(h, c.f)
			}
			for _, b := range h.CommandButtons(c.f) {
				if b.Menu && b.Pips > 0 {
					t.Errorf("%s [%s]: %s has pips where the menu mark sits", c.name, mode.name, b.ID)
				}
			}
		}
	}
}

func TestNoGridLabelClipsOnAnyPageInEitherMode(t *testing.T) {
	defer ui.SetNamer(nil)
	width := ui.MaxChars(CellW-4, 1)
	frames := map[string]Frame{"full": fullFrame(t), "harbor": harborFrame(t), "caesar": caesarFrame(t), "tower": towerCard(t)}
	for _, mode := range namesModes() {
		mode.apply()
		for name, f := range frames {
			for _, page := range []string{PageMain, PageMore, PageSummon, PageBuild} {
				h := New(nil, (&fakeRunner{}).backend())
				h.PageID = page
				for _, b := range h.CommandButtons(f) {
					if lost := labelLost(b.Label, width); lost != "" {
						t.Errorf("%s/%s [%s]: %s reads %q", name, page, mode.name, b.ID, lost)
					}
					for _, row := range CellLabel(b.Label, width) {
						if strings.Contains(row, "...") {
							t.Errorf("%s/%s [%s]: %s row %q is clipped", name, page, mode.name, b.ID, row)
						}
					}
				}
			}
		}
	}
}
