package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

type cardCase struct {
	name string
	f    Frame
}

func cardTextRows(h *HUD, f Frame) (int, int) {
	c := h.CardFor(f)
	box := PortraitBox()
	w := CardBox.X + CardBox.W - box.X - box.W - 14
	y := CardBox.Y + 6 + ui.LineHeight
	if c.K8s != "" {
		y += ui.LineHeight
	}
	room := (CardBottom(c, f) - y) / ui.LineHeight
	kept := h.FitLines(c.Lines, w, room)
	for _, l := range c.Lines {
		if !l.Optional && !slices.ContainsFunc(kept, func(k Line) bool { return k.Text == l.Text }) {
			return len(c.Lines) + room, room
		}
	}
	return h.LineRows(kept, w), room
}

func towerCard(t *testing.T) Frame {
	f := fullFrame(t)
	f.State.Reveals = append(f.State.Reveals, QosReveal)
	tower := view.Structure{ID: "keep-1", Kind: view.StructKeep, Variant: "tower", State: view.StateReady, Label: "The Lone Tower",
		Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}, Class: "balanced", Instance: "general-large", CPU: 2, MemoryGiB: 4, Territory: 11}
	f.Nodes = map[string]model.NodeView{"lone-tower": {SystemCPUMilli: 950, SystemMemBytes: 400 << 20}}
	return selectStructure(f, tower)
}

func cardCases(t *testing.T) []cardCase {
	stone := selectStructure(fullFrame(t), view.Structure{ID: "keep-2", Kind: view.StructKeep, Variant: "stone", State: view.StateReady, Label: "Stone Keep 2",
		Object: view.ObjectRef{Kind: "Node", Name: "stone-keep-2"}, Class: "balanced", Instance: "general-4xlarge", CPU: 16, MemoryGiB: 64, Territory: 12})
	stone.Metrics.Nodes["stone-keep-2"] = view.NodeMetrics{CPUAllocatableMilli: 15800, CPURequestedMilli: 3000, MemoryAllocatableBytes: 63 << 30, MemoryRequestedBytes: 4 << 30}
	pod := fullFrame(t)
	pod.State.Reveals = append(pod.State.Reveals, QosReveal)
	porter := pod.Pods[0]
	porter.QoS, porter.Focus, porter.People = model.QoSBurstable, "porter", []model.Kubernite{{Name: "porter"}}
	pod.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: porter}
	door := pod
	door.Selection = Selection{Kind: "hearth", ID: "default/porter-lone-tower", Pod: model.PodView{Namespace: "default", Name: "porter-lone-tower",
		Node: "lone-tower", Image: "porter:1.0", Profession: "Porter", State: model.HearthReady, Static: true, QoS: model.QoSBurstable}}
	depot := caesarFrame(t)
	depot = selectStructure(depot, view.Structure{ID: "guildhall-1", Kind: view.StructGuildhall, State: view.StateReady, Label: "Guildhall",
		Object: view.ObjectRef{Kind: "Service", Name: "parcels"}})
	codex := caesarFrame(t)
	codex = selectStructure(codex, view.Structure{ID: "codex-1", Kind: view.StructCodex, State: view.StateReady, Label: "The Codex",
		Object: view.ObjectRef{Kind: "Pod", Namespace: "kube-system", Name: "etcd-lone-tower"}})
	gateF, gateS := gateFrame(t)
	gate := selectStructure(gateF, gateS)
	return []cardCase{{"lone tower", towerCard(t)}, {"stone keep", stone}, {"porter", pod}, {"door", door}, {"depot", depot}, {"codex", codex}, {"gate", gate}}
}

func TestEveryCardLineAndButtonFits(t *testing.T) {
	for _, c := range cardCases(t) {
		h := New(nil, (&fakeRunner{}).backend())
		assertTextFits(t, h, c.f, textPanel{name: c.name + " card", bounds: fixed(CardBox),
			draw: (*HUD).DrawSelection, buttons: (*HUD).SelectionButtons})
		for _, mode := range namesModes() {
			mode.apply()
			if need, room := cardTextRows(h, c.f); need > room {
				t.Errorf("%s card [%s]: %d rows of text in room for %d", c.name, mode.name, need, room)
			}
		}
		ui.SetNamer(nil)
	}
}

func TestEveryCommandIsOneClickAwayWithoutPages(t *testing.T) {
	f := towerCard(t)
	h := New(nil, (&fakeRunner{}).backend())
	for _, a := range SelectionActions(f) {
		if gridCell(t, h, f, "act:"+a.Key) < 0 {
			t.Errorf("%s is not one click away: %v", a.Key, gridIDs(h, f))
		}
	}
	if gridCell(t, h, f, "act:more") >= 0 {
		t.Error("the grid needs no pager")
	}
	f.Guide.Target, f.Guide.Button = "keep:lone-tower", "act:"+KeyDescribe
	if b, ok := HitID(h.CommandButtons(f), "act:"+KeyDescribe); !ok || !b.Selected {
		t.Fatalf("the guide marks its cell: %+v", b)
	}
}

func TestFitLinesDropsFlavourFirst(t *testing.T) {
	h := New(nil, Backend{})
	lines := []Line{{Text: "A"}, {Text: "flavour one", Optional: true}, {Text: "B"}, {Text: "flavour two", Optional: true}}
	if got := h.FitLines(lines, 180, 3); len(got) != 3 || got[1].Text != "flavour one" || got[2].Text != "B" {
		t.Fatalf("fit = %+v", got)
	}
	if got := h.FitLines(lines, 180, 1); len(got) != 2 || got[0].Text != "A" || got[1].Text != "B" {
		t.Fatalf("essential lines stay even when crowded: %+v", got)
	}
	if got := h.FitLines(lines, 180, 9); len(got) != 4 {
		t.Fatal("room for all keeps all")
	}
}
